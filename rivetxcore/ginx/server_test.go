package ginx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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
