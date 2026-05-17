package ginx

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"rivetxgo/rivetxcore/log"
	"rivetxgo/rivetxcore/session"
	"rivetxgo/rivetxcore/utilx"
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
}

func HttpJsonResp(c *gin.Context, isReadBody bool, doFunc func(req *HttpReqData) (interface{}, error)) {
	sessionId := session.SessionId()
	currTime := time.Now()
	elapsed := int64(0)
	req := &HttpReqData{C: c}
	var data map[string]interface{}
	value, err := func() (ret interface{}, retErr error) {
		defer func() {
			endTime := time.Now()
			elapsed = endTime.UnixMilli() - currTime.UnixMilli()
		}()

		defer func() {
			if r := recover(); r != nil {
				retErr = ee.New(nil, "sessionId:%v, panic:%v, Stack trace:%s \n", sessionId, r, string(debug.Stack()))
				ret = nil
			}
		}()
		if isReadBody {
			Body, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				return nil, ee.New(err, "read req_body")
			}
			req.Body = Body
		}
		return doFunc(req)
	}()

	if req.IsDisablePrintBody {
		req.Body = nil
	}

	if err != nil {
		data = map[string]interface{}{
			"success":   false,
			"code":      -1,
			"msg":       fmt.Sprintf("%v", err),
			"elapsed":   elapsed,
			"sessionId": sessionId,
		}
		log4.Error("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%+v",
			sessionId, currTime.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
			utilx.SliceByteToString(req.Body), data)
	} else {
		data = map[string]interface{}{
			"success":   true,
			"code":      0,
			"msg":       "",
			"result":    value,
			"elapsed":   elapsed,
			"sessionId": sessionId,
		}
		if !req.IsDisableAccessLog {
			if log.LogHttp().GetLevel() == log4.INFO {
				data := map[string]interface{}{
					"success":   true,
					"code":      0,
					"msg":       "",
					"elapsed":   elapsed,
					"sessionId": sessionId,
				}
				log.LogHttp().Info("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%v",
					sessionId, currTime.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
					nil, data)
			} else if log.LogHttp().GetLevel() < log4.INFO {
				printData, _ := json.Marshal(data)
				log.LogHttp().Debug("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%+v",
					sessionId, currTime.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
					utilx.SliceByteToString(req.Body), utilx.SliceByteToString(printData))
			}
		} else {
			if log.LogHttp().GetLevel() == log4.TRACE {
				printData, _ := json.Marshal(data)
				log.LogHttp().Trace("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%+v",
					sessionId, currTime.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
					utilx.SliceByteToString(req.Body), utilx.SliceByteToString(printData))
			} else {
				data := map[string]interface{}{
					"success":   true,
					"code":      0,
					"msg":       "",
					"elapsed":   elapsed,
					"sessionId": sessionId,
				}
				log.LogHttp().Info("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%v",
					sessionId, currTime.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
					nil, data)
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log4.Error("accessLog sessionId:%v, startTime:[%s UTC], elapsed:%v, Method:%v, RemoteIP:%v, url:%v, header:%v, req_body:%v, outData:%+v",
			sessionId, currTime.UTC().Format("2006-01-02 15:04:05.000"), elapsed, c.Request.Method, c.RemoteIP(), c.Request.URL, c.Request.Header,
			utilx.SliceByteToString(req.Body), data)
		data = map[string]interface{}{
			"success":   false,
			"code":      -1,
			"msg":       fmt.Sprintf("json.Marshal err:%v", err),
			"elapsed":   elapsed,
			"sessionId": sessionId,
		}
		jsonData, err = json.Marshal(data)
		if err != nil {
			jsonData = []byte(fmt.Sprintf("json.Marshal err:%v", err))
		}
	}

	n, err := c.Writer.Write(jsonData)
	if err != nil {
		log4.Error("client connection lost during write, sessionId:%v, expected:%v, bytesWritten:%v, err:%v",
			sessionId, len(jsonData), n, err)
		return
	}

	if n < len(jsonData) {
		log4.Warn("short write detected, sessionId:%v, expected:%v, actual:%v",
			sessionId, len(jsonData), n)
	}
}
