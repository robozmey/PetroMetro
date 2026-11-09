package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRepository struct {
	err  error
	slug string
}

func (f *fakeRepository) ListCities(context.Context) ([]domain.City, error) {
	return []domain.City{}, f.err
}
func (f *fakeRepository) GetSchemaBySlug(_ context.Context, slug string) (domain.MetroSchema, error) {
	f.slug = slug
	return domain.MetroSchema{City: domain.City{ID: 1, Name: "City", Slug: slug}, Stations: []domain.Station{{ID: 1, Year: 1982}}, Lines: []domain.MetroLine{}}, f.err
}
func TestSchemaHTTP(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"success", nil, 200}, {"missing", repository.ErrNotFound, 404}, {"database failure", errors.New("private database details"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{err: tc.err}
			app := application{repo: repo}
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/cities/{citySlug}/metro/schema", app.schemaHandler)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/cities/example/metro/schema", nil))
			if w.Code != tc.status || repo.slug != "example" {
				t.Fatalf("status=%d slug=%s", w.Code, repo.slug)
			}
			if tc.err == nil {
				var schema domain.MetroSchema
				if err := json.Unmarshal(w.Body.Bytes(), &schema); err != nil {
					t.Fatal(err)
				}
				if schema.City.Slug != "example" || schema.Stations[0].Year != 1982 || schema.Lines == nil {
					t.Fatal("response contract changed")
				}
			} else {
				var body map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["error"] == "" || body["error"] == tc.err.Error() {
					t.Fatal("invalid public error")
				}
			}
		})
	}
}
