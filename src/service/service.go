package service

import (
	"context"
	"os"

	"github.com/jenyasd209/fake-sensors/src/api"
	"github.com/jenyasd209/fake-sensors/src/generator"
	"github.com/jenyasd209/fake-sensors/src/storage"
)

type Service struct {
	generator *generator.Generator
	apiServer *api.Server
}

func NewService() (*Service, error) {
	s, err := storage.NewStorage(
		storage.WithDbUser(os.Getenv("POSTGRES_USER")),
		storage.WithDbPassword(os.Getenv("POSTGRES_PASSWORD")),
		storage.WithDbPort(os.Getenv("POSTGRES_PORT")),
		storage.WithDbHost(os.Getenv("POSTGRES_HOST")),
		storage.WithDbName(os.Getenv("POSTGRES_NAME")),
		storage.WithRedisAddress(os.Getenv("REDIS_ADDRESS")),
	)
	if err != nil {
		panic(err)
	}

	g, err := generator.NewGenerator(s)
	if err != nil {
		panic(err)
	}

	return &Service{
		generator: g,
		apiServer: api.DefaultApiServer(s),
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	err := s.generator.Start(ctx)
	if err != nil {
		panic(err)
	}

	defer s.generator.Stop()

	return s.apiServer.Run(":8080")
}
