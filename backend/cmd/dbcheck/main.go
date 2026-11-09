package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robozmey/podzemka/backend/internal/repository"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := repository.OpenPostgres(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)

	cities, err := repo.ListCities(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("cities: %d\n", len(cities))
	for _, city := range cities {
		schema, err := repo.GetSchemaBySlug(ctx, city.Slug)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d %s (%s): %d stations, %d lines\n", city.ID, city.Name, city.Slug, len(schema.Stations), len(schema.Lines))
	}
}
