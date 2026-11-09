package main

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

//go:embed data/*-stations.json
var schemaData embed.FS

var cities = []domain.City{
	{ID: 1, Name: "Санкт-Петербург", Slug: "saint-petersburg"},
	{ID: 2, Name: "Москва", Slug: "moscow"},
	{ID: 3, Name: "Красноярск", Slug: "krasnoyarsk"},
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := repository.OpenPostgres(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	for _, city := range cities {
		schema := loadSchema(city)
		if err := repo.SaveSchema(ctx, schema); err != nil {
			log.Fatal(err)
		}
		log.Printf("saved %s: %d stations, %d lines", city.Slug, len(schema.Stations), len(schema.Lines))
	}
}

func loadSchema(city domain.City) domain.MetroSchema {
	data, err := schemaData.ReadFile("data/" + city.Slug + "-stations.json")
	if err != nil {
		log.Fatalf("read %s station data: %v", city.Slug, err)
	}

	var schema domain.MetroSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		log.Fatalf("parse %s station data: %v", city.Slug, err)
	}

	schema.City = city
	return schema
}
