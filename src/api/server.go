package api

import (
	"github.com/jenyasd209/fake-sensors/src/api/routes"
	"github.com/jenyasd209/fake-sensors/src/storage"
)

type Server struct {
	router *routes.Router
}

func DefaultApiServer(storage *storage.Storage) *Server {
	return &Server{router: routes.NewRouter(storage)}
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
