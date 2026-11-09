package main

import (
	"context"
	"encoding/json"
	"errors"
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

type application struct{ repo schemaRepository }

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

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Hello, World!"))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (app application) citiesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cities, err := app.repo.ListCities(ctx)
	if err != nil {
		log.Printf("list cities: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось загрузить города"})
		return
	}
	writeJSON(w, http.StatusOK, cities)
}

func (app application) schemaHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	schema, err := app.repo.GetSchemaBySlug(ctx, r.PathValue("citySlug"))
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Схема города не найдена"})
		return
	}

	if err != nil {
		log.Printf("get schema: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось загрузить схему"})
		return
	}
	writeJSON(w, http.StatusOK, schema)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
