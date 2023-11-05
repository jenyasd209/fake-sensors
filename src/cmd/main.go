package main

import (
	"context"

	"github.com/jenyasd209/fake-sensors/src/service"
)

func main() {
	s, err := service.NewService()
	if err != nil {
		panic(err)
	}

	err = s.Start(context.TODO())
	if err != nil {
		panic(err)
	}
}
