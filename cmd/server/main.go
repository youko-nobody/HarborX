package main

import (
	"log"
	"net/http"

	"harborx/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    application.Config.ListenAddress(),
		Handler: application.Router,
	}

	log.Printf("harborx listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
