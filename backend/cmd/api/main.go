package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/overpass"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

type schemaRepository interface {
	ListCities(context.Context) ([]domain.City, error)
	GetSchemaBySlug(context.Context, string) (domain.MetroSchema, error)
	SaveSchema(context.Context, domain.MetroSchema) error
	CityExists(context.Context, string) (bool, error)
}

type application struct{ repo schemaRepository }

type createCityRequest struct {
	Name     string             `json:"name"`
	Slug     string             `json:"slug"`
	Stations []domain.Station   `json:"stations"`
	Lines    []domain.MetroLine `json:"lines"`
}

var validCitySlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

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
	mux.HandleFunc("POST /api/cities", app.createCityHandler)
	mux.HandleFunc("POST /api/cities/preview-import", app.previewImportHandler)
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

func (app application) previewImportHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не удалось прочитать файл Overpass (максимум 10 МБ)"})
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Загрузите JSON-файл Overpass в поле file"})
		return
	}
	defer file.Close()

	schema, err := overpass.Parse(file, overpass.MapWidth, overpass.MapHeight)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не удалось разобрать файл Overpass"})
		return
	}
	if err := schema.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Парсер сформировал некорректную схему"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"stationsCount": len(schema.Stations),
		"linesCount":    len(schema.Lines),
		"schema":        schema,
	})
}

func (app application) createCityHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request createCityRequest
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Некорректное тело запроса"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "В запросе должен быть один JSON-объект"})
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Slug = strings.TrimSpace(request.Slug)
	if request.Name == "" || !validCitySlug.MatchString(request.Slug) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Укажите название города и slug в формате lowercase-hyphen"})
		return
	}

	content := domain.MetroContent{Stations: request.Stations, Lines: request.Lines}
	if err := content.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Некорректная схема метро"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	exists, err := app.repo.CityExists(ctx, request.Slug)
	if err != nil {
		log.Printf("check city before create: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось проверить город"})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Город с таким slug уже существует"})
		return
	}

	schema := domain.MetroSchema{
		City:     domain.City{Name: request.Name, Slug: request.Slug},
		Stations: content.Stations,
		Lines:    content.Lines,
	}
	if err := app.repo.SaveSchema(ctx, schema); err != nil {
		log.Printf("save city schema: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось сохранить город"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"name": request.Name, "slug": request.Slug})
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
