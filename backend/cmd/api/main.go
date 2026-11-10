package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

type schemaRepository interface {
	ListCities(context.Context) ([]domain.City, error)
	GetSchemaBySlug(context.Context, string) (domain.MetroSchema, error)
}

type application struct {
	repo schemaRepository
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := repository.OpenPostgres(ctx)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := application{repo: repository.NewPostgresRepository(db)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", helloHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/cities", app.citiesHandler)
	mux.HandleFunc("GET /api/cities/{citySlug}/metro/schema", app.schemaHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("API is running on http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}
