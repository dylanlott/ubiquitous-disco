package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/dylanlott/ubiquitous-disco/pkg/server"
)

//go:embed static/*
var static embed.FS

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

func main() {
	var addr string
	port := os.Getenv("PORT")
	if port == "" {
		addr = ":8080"
	} else {
		addr = fmt.Sprintf(":%s", port)
	}

	// make a new server with templates and a listener address
	srv, err := server.New(t, static, addr)
	if err != nil {
		log.Fatalf("failed to create new server: %s", err)
	}

	log.Fatalf("fatal server error: %s", srv.Serve())
}
