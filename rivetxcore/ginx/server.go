package ginx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yefy/rivetxgo/rivetxcore/config"
	"github.com/yefy/rivetxgo/rivetxcore/log"
	"github.com/yefy/rivetxgo/rivetxcore/session"
	"github.com/yefy/rivetxgo/rivetxcore/utilx"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
)

func NewServer(isOpenGinLog bool) *Server {
	server := &Server{}
	server.router = server.NewRouter(isOpenGinLog)
	return server
}

type Server struct {
	router     *gin.Engine
	httpServer *http.Server
}

func (server *Server) NewRouter(isOpenGinLog bool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	if isOpenGinLog {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())

	// register pprof routes
	pprof.Register(r)

	return r
}

func (server *Server) Router() *gin.Engine {
	return server.router
}

func (server *Server) Listen(Addr string) error {
	server.httpServer = &http.Server{
		Addr:         Addr,
		Handler:      server.router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  150 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			if tc, ok := c.(*net.TCPConn); ok {
				if err := tc.SetNoDelay(true); err != nil {
					log4.Error("Error setting TCP_NODELAY:%v", err)
				} else {
					//log4.Info("TCP_NODELAY set to true for connection:%v", c.RemoteAddr())
				}
			}
			return ctx
		},
	}

	log4.Info("start listen addr=%s", Addr)
	err := server.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return ee.New(err, "err:listen addr=%s", Addr)
	}
	log4.Info("close listen addr=%s", Addr)
	return nil
}

func (server *Server) Stop() {
	if server.httpServer != nil {
		httpCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer httpCancel()
		err := server.httpServer.Shutdown(httpCtx)
		if err != nil {
			log4.Error("Error shutting down HTTP server:%v", err)
		}
	}
}

type HttpReqData struct {
	C                  *gin.Context
	IsDisableAccessLog bool
	IsDisablePrintBody bool
	Body               []byte
	Status             int
	Code               int
	DesMsg             string
	DataMsg            any
}

type HttpJsonOptions struct {
	MaxBodyBytes int64
	// FormatData 在业务回调和 DataMsg 覆盖之后统一转换响应数据，也处理错误和空结果。
	FormatData func(value any) any
	// FormatLogBody 可选；接收请求体副本，返回仅供日志打印的内容。具体格式和字段规则由调用方决定。
	FormatLogBody func(body []byte) []byte
}

// formatLogBody 隔离日志回调与业务输入；日志格式化失败不能改变已经执行的业务及 HTTP 结果。
func formatLogBody(body []byte, format func([]byte) []byte) (result []byte) {
	if format == nil {
		return body
	}
	defer func() {
		if recover() != nil {
			result = []byte("[request log body formatter failed]")
			log4.Error("access log request body formatter failed")
		}
	}()
	return format(bytes.Clone(body))
}

// HttpJsonResp keeps the callback return value as the default response data.
// DataMsg can override it, including when the callback returns an error.
func HttpJsonResp(c *gin.Context, isReadBody bool, doFunc func(req *HttpReqData) (interface{}, error), options ...HttpJsonOptions) {
	sessionID := session.SessionId()
	started := time.Now()
	req := &HttpReqData{C: c}
	var option HttpJsonOptions
	if len(options) > 0 {
		option = options[0]
	}
	maxBodyBytes := int64(1 << 20)
	if option.MaxBodyBytes > 0 {
		maxBodyBytes = option.MaxBodyBytes
	}

	value, callErr := func() (ret interface{}, retErr error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				retErr = ee.New(nil, "sessionId:%v, panic:%v, Stack trace:%s \n", sessionID, recovered, string(debug.Stack()))
				ret = nil
			}
		}()
		if isReadBody {
			body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes))
			if err != nil {
				return nil, ee.New(err, "read req_body")
			}
			req.Body = body
		}
		return doFunc(req)
	}()

	if callErr == nil && req.DataMsg != nil {
		log4.Error("HttpJsonResp: DataMsg set on successful callback, path=%s", c.Request.URL.Path)
	}
	if req.DataMsg != nil {
		value = req.DataMsg
	}
	// 响应结构由调用方在 HTTP 边界统一；不改变回调使用的业务对象或持久化结构。
	if option.FormatData != nil {
		value = option.FormatData(value)
	}
	code := req.Code
	if callErr != nil && code == 0 {
		code = -1
	}
	description := req.DesMsg
	if description == "" && callErr != nil {
		if config.IsOpenStackInfoToErrorLog() {
			description = fmt.Sprintf("%+#v", callErr)
		} else {
			description = callErr.Error()
		}
	}
	elapsed := time.Since(started).Milliseconds()
	data := map[string]interface{}{
		"success":   code >= 0,
		"code":      code,
		"des_msg":   description,
		"data_msg":  value,
		"elapsed":   elapsed,
		"sessionId": sessionID,
	}
	// 仅在现有日志规则需要打印请求体时调用格式化回调；成功与错误日志共用同一份日志内容。
	// 原始 req.Body 始终保留给业务，框架不解析字段、不内置任何业务脱敏规则。
	var logBody []byte
	if !req.IsDisablePrintBody && (code < 0 || (!req.IsDisableAccessLog && log.LogHttp().GetLevel() < log4.INFO)) {
		logBody = formatLogBody(req.Body, option.FormatLogBody)
	}
	if code < 0 {
		log4.Error("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%+v",
			sessionID, started.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
			utilx.SliceByteToString(logBody), data)
	} else if !req.IsDisableAccessLog {
		if log.LogHttp().GetLevel() < log4.INFO {
			printData, _ := json.Marshal(data)
			log.LogHttp().Debug("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%+v",
				sessionID, started.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
				utilx.SliceByteToString(logBody), utilx.SliceByteToString(printData))
		} else {
			log.LogHttp().Info("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v",
				sessionID, started.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
				nil)
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log4.Error("HttpJsonResp: marshal response: %v", err)
		var emptyData any
		if option.FormatData != nil {
			emptyData = option.FormatData(nil)
		}
		jsonData, _ = json.Marshal(map[string]interface{}{
			"success": false, "code": -1, "des_msg": fmt.Sprintf("json.Marshal err:%v", err),
			"data_msg": emptyData, "elapsed": elapsed, "sessionId": sessionID,
		})
	}
	status := req.Status
	if status == 0 {
		status = http.StatusOK
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Status(status)
	if _, err := c.Writer.Write(jsonData); err != nil {
		log4.Error("client connection lost during write, sessionId:%v, err:%v", sessionID, err)
	}
}
