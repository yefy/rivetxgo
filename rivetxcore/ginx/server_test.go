package ginx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yefy/log4go/log4"
)

func runJSONHandler(t *testing.T, body string, readBody bool, options HttpJsonOptions, callback func(*HttpReqData) (interface{}, error)) (int, map[string]any, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		HttpJsonResp(c, readBody, callback, options)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	var result map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return recorder.Code, result, recorder.Header().Get("Content-Type")
}

func TestHttpJsonRespDefaultResult(t *testing.T) {
	status, result, contentType := runJSONHandler(t, "", false, HttpJsonOptions{}, func(req *HttpReqData) (interface{}, error) {
		req.IsDisableAccessLog = true
		return map[string]string{"id": "one"}, nil
	})
	if status != http.StatusOK || contentType != "application/json; charset=utf-8" ||
		result["success"] != true || result["code"] != float64(0) || result["des_msg"] != "" {
		t.Fatalf("unexpected response: status=%d content-type=%q body=%v", status, contentType, result)
	}
	data := result["data_msg"].(map[string]any)
	if data["id"] != "one" {
		t.Fatalf("callback result was lost: %v", result)
	}
}

func TestHttpJsonRespRequestFieldsOverrideErrorResult(t *testing.T) {
	status, result, _ := runJSONHandler(t, "", false, HttpJsonOptions{}, func(req *HttpReqData) (interface{}, error) {
		req.Status = http.StatusServiceUnavailable
		req.Code = -7
		req.DesMsg = "retry later"
		req.DataMsg = map[string]string{"id": "saved"}
		return map[string]string{"id": "ignored"}, errors.New("internal cause")
	})
	if status != http.StatusServiceUnavailable || result["success"] != false ||
		result["code"] != float64(-7) || result["des_msg"] != "retry later" {
		t.Fatalf("request fields were not applied: %v", result)
	}
	if result["data_msg"].(map[string]any)["id"] != "saved" {
		t.Fatalf("DataMsg did not override callback data: %v", result)
	}
}

func TestHttpJsonRespBodyLimitStopsCallback(t *testing.T) {
	called := false
	status, result, _ := runJSONHandler(t, "12345", true, HttpJsonOptions{MaxBodyBytes: 4}, func(req *HttpReqData) (interface{}, error) {
		called = true
		return nil, nil
	})
	if called || status != http.StatusOK || result["success"] != false || result["code"] != float64(-1) {
		t.Fatalf("body limit was not applied: called=%v status=%d result=%v", called, status, result)
	}
}

func TestHttpJsonRespSuccessfulDataOverride(t *testing.T) {
	_, result, _ := runJSONHandler(t, "", false, HttpJsonOptions{}, func(req *HttpReqData) (interface{}, error) {
		req.IsDisableAccessLog = true
		req.Code = 2
		req.DataMsg = "preferred"
		return "fallback", nil
	})
	if result["success"] != true || result["code"] != float64(2) || result["data_msg"] != "preferred" {
		t.Fatalf("successful DataMsg override was not applied: %v", result)
	}
}

func TestHttpJsonRespExplicitCodeOverridesCallbackError(t *testing.T) {
	_, result, _ := runJSONHandler(t, "", false, HttpJsonOptions{}, func(req *HttpReqData) (interface{}, error) {
		req.Code = 3
		req.DesMsg = "accepted with warning"
		return "accepted", errors.New("callback warning")
	})
	if result["code"] != float64(3) || result["success"] != true || result["des_msg"] != "accepted with warning" {
		t.Fatalf("explicit request fields were ignored: %v", result)
	}
}

// 通过真实文件日志验证回调只改变 access log，不改变业务输入、HTTP 结果或现有日志级别规则。
func TestHttpJsonRespFormatLogBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	logger := log4.NewLog4("")
	if err := logger.Run(&log4.Log4Config{
		RefreshRate: 3600,
		Appenders:   map[string]log4.Log4ConfigAppender{"file": {Kind: log4.KindFile, Path: path, Pattern: "%M"}},
		Root:        log4.Log4ConfigLogger{Level: "debug", Appenders: []string{"file"}},
		Loggers:     map[string]log4.Log4ConfigLogger{"http": {Level: "debug", Appenders: []string{"file"}}},
	}); err != nil {
		t.Fatal(err)
	}
	closeLogger := func() {
		if logger.CloseCompareAndSwap() {
			logger.Close(true)
		}
	}
	t.Cleanup(func() {
		closeLogger()
		log4.GLog4.Store(log4.NewLog4(""))
	})

	for _, tc := range []struct {
		name                                                                  string
		failed, info, disableAccess, disableBody, noFormatter, panicFormatter bool
		wantCall                                                              bool
	}{
		{name: "success", wantCall: true},
		{name: "failure", failed: true, wantCall: true},
		{name: "error-still-logged", failed: true, disableAccess: true, wantCall: true},
		{name: "info", info: true},
		{name: "disabled-access", disableAccess: true},
		{name: "disabled-body", disableBody: true},
		{name: "no-formatter", noFormatter: true},
		{name: "panic", panicFormatter: true, wantCall: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger.Target("http").Level = log4.DEBUG
			if tc.info {
				logger.Target("http").Level = log4.INFO
			}
			original := "raw-" + tc.name
			called := false
			option := HttpJsonOptions{FormatLogBody: func(body []byte) []byte {
				called = true
				if string(body) != original {
					t.Error("formatter did not receive request body")
				}
				// 模拟调用方原地修改副本，原始业务输入仍必须保留。
				for i := range body {
					body[i] = 'x'
				}
				if tc.panicFormatter {
					panic("formatter fault")
				}
				return []byte("logged-" + tc.name)
			}}
			if tc.noFormatter {
				option.FormatLogBody = nil
			}
			var request *HttpReqData
			status, response, _ := runJSONHandler(t, original, true, option, func(req *HttpReqData) (interface{}, error) {
				request = req
				req.IsDisableAccessLog, req.IsDisablePrintBody = tc.disableAccess, tc.disableBody
				if tc.failed {
					return nil, errors.New("business failed")
				}
				return "accepted", nil
			})
			if status != http.StatusOK || response["success"] != !tc.failed || called != tc.wantCall || string(request.Body) != original {
				t.Fatal("logging changed business input, response or callback rules")
			}
		})
	}
	closeLogger()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{"logged-success", "logged-failure", "logged-error-still-logged", "raw-no-formatter", "[request log body formatter failed]"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing access log %q", want)
		}
	}
	for _, unwanted := range []string{"raw-success", "raw-failure", "raw-error-still-logged", "raw-info", "raw-disabled-access", "raw-disabled-body", "raw-panic"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("unexpected original body %q in log", unwanted)
		}
	}
}
