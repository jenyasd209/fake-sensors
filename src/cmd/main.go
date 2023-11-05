package main

import (
	"context"
	"os"

	"github.com/jenyasd209/fake-sensors/src/api"
	"github.com/jenyasd209/fake-sensors/src/generator"
	"github.com/jenyasd209/fake-sensors/src/storage"
)

func main() {
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

	err = g.Start(context.TODO())
	if err != nil {
		panic(err)
	}

	server := api.DefaultApiServer(s)
	err = server.Run(":8080")
	if err != nil {
		panic(err)
	}
}
