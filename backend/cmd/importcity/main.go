package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

func main() {
	file := flag.String("file", "", "путь к JSON-файлу схемы")
	name := flag.String("name", "", "название города")
	slug := flag.String("slug", "", "slug города")
	flag.Parse()

	if *file == "" || *name == "" || *slug == "" {
		log.Fatal("необходимо указать путь к JSON-файлу, название города и slug города")
	}

	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}

	var content domain.MetroContent
	if err := json.Unmarshal(data, &content); err != nil {
		log.Fatal(err)
	}

	if err := content.Validate(); err != nil {
		log.Fatal(err)
	}

	schema := domain.MetroSchema{
		City: domain.City{
			Name: *name,
			Slug: *slug,
		},
		Stations: content.Stations,
		Lines:    content.Lines,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := repository.OpenPostgres(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)

	if err := repo.SaveSchema(ctx, schema); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Город %s успешно сохранен\n", *name)

}
