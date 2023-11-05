package main

import (
	"context"

	"github.com/jenyasd209/fake-sensors/src/api"
	"github.com/jenyasd209/fake-sensors/src/generator"
	"github.com/jenyasd209/fake-sensors/src/storage"
)

func main() {
	s, err := storage.NewStorage()
	if err != nil {
		panic(err)
	}

	g, err := generator.NewGenerator(s)
	if err != nil {
		panic(err)
	}
	g.Start(context.TODO())

	server := api.DefaultApiServer(s)
	err = server.Run(":8080")
	if err != nil {
		panic(err)
	}
}
