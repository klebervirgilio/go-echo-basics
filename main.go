package main

import (
	"github.com/klebervirgilio/go-echo-basics/config"
	"github.com/klebervirgilio/go-echo-basics/http"
	mongorepository "github.com/klebervirgilio/go-echo-basics/storage"
)

func main() {
	cfg := config.New()
	repository := mongorepository.NewMongoRepo(cfg)
	server := http.NewServer(repository, cfg)
	server.Serve()
}
