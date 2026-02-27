package httpapi

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

func registerPprofRoutes(rg *gin.RouterGroup) {
	pp := rg.Group("/pprof")
	{
		pp.GET("/", gin.WrapF(pprof.Index))
		pp.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pp.GET("/profile", gin.WrapF(pprof.Profile))
		pp.GET("/symbol", gin.WrapF(pprof.Symbol))
		pp.POST("/symbol", gin.WrapF(pprof.Symbol))
		pp.GET("/trace", gin.WrapF(pprof.Trace))

		pp.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		pp.GET("/block", gin.WrapH(pprof.Handler("block")))
		pp.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		pp.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		pp.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		pp.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
}
