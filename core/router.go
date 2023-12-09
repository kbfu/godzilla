package core

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"godzilla/chaos"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()
	pprof.Register(router)

	//repo := router.Group("/repo")
	chaosGrp := router.Group("/chaos")

	chaosGrp.POST("/create", chaos.CreateChaos)
	return router
}
