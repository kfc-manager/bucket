package main

import (
	"fmt"
	"os"

	"github.com/kfc-manager/bucket/domain"
	"github.com/kfc-manager/bucket/server"
)

func main() {
	auth := domain.NewAuth(
		envOrPanic("ACCESS_KEY"),
		envOrPanic("SECRET_KEY"),
	)
	storage, err := domain.NewStorage("./data")
	if err != nil {
		panic(err)
	}

	if err := server.New("8000", auth, storage).Listen(); err != nil {
		panic(err)
	}
}

func envOrPanic(key string) string {
	value := os.Getenv(key)
	if len(value) < 1 {
		panic(fmt.Errorf("environment variable '%s' is missing", key))
	}
	return value
}
