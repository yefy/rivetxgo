package examples

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/yefy/log4go/log4"
	"rivetxgo/rivetxcore/ginx"
	"rivetxgo/rivetxcore/gox"
)

func LogReopenTests() {
	gox.Spawn(func(u uint64) error {
		server := ginx.NewServer(true)
		defer server.Stop()

		router := server.Router()
		{
			//curl http://127.0.0.1:48080/test/ok200 -k -v
			r := router.Group("/test")
			r.GET("/ok200", func(c *gin.Context) {
				_, _ = c.Writer.Write([]byte(`{code=0}`))
			})
		}

		{
			//curl http://127.0.0.1:48080/log/reopen -k -v
			r := router.Group("/log")
			r.GET("/reopen", func(c *gin.Context) {
				ginx.HttpJsonResp(c, true, func(req *ginx.HttpReqData) (interface{}, error) {
					log4.Reopen()
					return nil, nil
				})
			})
		}
		return server.Listen(fmt.Sprintf("0.0.0.0:%d", 48080))
	})
}
