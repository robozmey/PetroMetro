package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/robozmey/podzemka/backend/internal/overpass"
)

func main() {
	inputPath := flag.String("input", "", "путь к JSON-выгрузке Overpass (обязательный)")
	outputPath := flag.String("output", "", "путь для итогового JSON; по умолчанию stdout")
	width := flag.Float64("width", overpass.MapWidth, "ширина схемы в координатах SVG")
	height := flag.Float64("height", overpass.MapHeight, "высота схемы в координатах SVG")
	flag.Parse()

	if *inputPath == "" {
		fatal(errors.New("укажите файл выгрузки через -input"))
	}
	input, err := os.Open(*inputPath)
	if err != nil {
		fatal(fmt.Errorf("открыть входной файл: %w", err))
	}
	defer input.Close()

	schema, err := overpass.Parse(input, *width, *height)
	if err != nil {
		fatal(err)
	}

	var output = os.Stdout
	if *outputPath != "" {
		file, err := os.Create(*outputPath)
		if err != nil {
			fatal(fmt.Errorf("создать файл результата: %w", err))
		}
		defer file.Close()
		output = file
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		fatal(fmt.Errorf("записать JSON: %w", err))
	}
	fmt.Fprintf(os.Stderr, "Импортировано станций: %d; линий: %d\n", len(schema.Stations), len(schema.Lines))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Ошибка:", err)
	os.Exit(1)
}
