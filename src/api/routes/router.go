package routes

import (
	_ "github.com/jenyasd209/fake-sensors/src/api/doc"
	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	routes  *gin.Engine
	storage *storage.Storage
}

func NewRouter(storage *storage.Storage) *Router {
	r := &Router{
		routes:  gin.Default(),
		storage: storage,
	}

	RegisterGroupRoutes(r)
	RegisterSensorRoutes(r)
	RegisterTemperatureRoutes(r)

	r.routes.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return r
}

func (r *Router) Run(addr string) error {
	return r.routes.Run(addr)
}
