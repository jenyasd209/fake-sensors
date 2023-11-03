package api

import (
	"github.com/jenyasd209/fake-sensors/src/api/routes"
	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

type Server struct {
	routes  *gin.Engine
	storage *storage.Storage
}

func DefaultApiServer(storage *storage.Storage) *Server {
	return &Server{
		routes:  gin.Default(),
		storage: storage,
	}
}

func (s *Server) Run(addr string) error {
	s.registerRoutes()
	return s.routes.Run(addr)
}

func (s *Server) registerRoutes() {
	routes.RegisterGroupRoutes(s.routes, s.storage)
	routes.RegisterSensorRoutes(s.routes, s.storage)
	routes.RegisterTemperatureRoutes(s.routes, s.storage)
}
