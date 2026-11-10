package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/robozmey/podzemka/backend/internal/repository"
)

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
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось загрузить города",
		})
		return
	}

	writeJSON(w, http.StatusOK, cities)
}

func (app application) schemaHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	schema, err := app.repo.GetSchemaBySlug(ctx, r.PathValue("citySlug"))
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Схема города не найдена",
		})
		return
	}

	if err != nil {
		log.Printf("get schema: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось загрузить схему",
		})
		return
	}

	writeJSON(w, http.StatusOK, schema)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
