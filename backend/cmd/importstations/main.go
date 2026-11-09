package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/robozmey/podzemka/backend/internal/domain"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	mapWidth  = 700.0
	mapHeight = 420.0
	mapMargin = 30.0
)

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Type string            `json:"type"`
	ID   int64             `json:"id"`
	Lat  float64           `json:"lat"`
	Lon  float64           `json:"lon"`
	Tags map[string]string `json:"tags"`
}

type outputStation = domain.Station
type outputLine = domain.MetroLine
type outputSchema = domain.MetroContent

type importedStation struct {
	osmID int64
	name  string
	lat   float64
	lon   float64
	year  int
	color string
}

var lineColors = []struct {
	name  string
	color string
}{
	{name: "Красная", color: "#e53935"},
	{name: "Синяя", color: "#1976d2"},
	{name: "Зелёная", color: "#2e9d50"},
	{name: "Оранжевая", color: "#f28c28"},
	{name: "Фиолетовая", color: "#8e44ad"},
	{name: "Коричневая", color: "#795548"},
}

func main() {
	inputPath := flag.String("input", "", "путь к JSON-выгрузке Overpass (обязательный)")
	outputPath := flag.String("output", "", "путь для итогового JSON; по умолчанию stdout")
	width := flag.Float64("width", mapWidth, "ширина схемы в координатах SVG")
	height := flag.Float64("height", mapHeight, "высота схемы в координатах SVG")
	flag.Parse()

	if *inputPath == "" {
		fatal(errors.New("укажите файл выгрузки через -input"))
	}
	if *width <= 2*mapMargin || *height <= 2*mapMargin {
		fatal(errors.New("ширина и высота должны быть больше двойного отступа"))
	}

	input, err := os.Open(*inputPath)
	if err != nil {
		fatal(fmt.Errorf("открыть входной файл: %w", err))
	}
	defer input.Close()

	stations, err := parseStations(input)
	if err != nil {
		fatal(err)
	}

	schema := buildSchema(stations, *width, *height)
	var output io.Writer = os.Stdout
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
	fmt.Fprintf(os.Stderr, "Импортировано станций: %d; линий с определённым цветом: %d\n", len(schema.Stations), len(schema.Lines))
}

func parseStations(input io.Reader) ([]importedStation, error) {
	var response overpassResponse
	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("разобрать Overpass JSON: %w", err)
	}

	unique := make(map[int64]importedStation)
	for _, element := range response.Elements {
		if element.Type != "node" || element.ID == 0 || element.Lat == 0 || element.Lon == 0 {
			continue
		}
		if element.Tags["station"] != "subway" && element.Tags["subway"] != "yes" {
			continue
		}
		name := strings.TrimSpace(element.Tags["name:ru"])
		if name == "" {
			name = strings.TrimSpace(element.Tags["name"])
		}
		if name == "" {
			continue
		}

		year := parseYear(element.Tags["start_date"])
		unique[element.ID] = importedStation{
			osmID: element.ID,
			name:  name,
			lat:   element.Lat,
			lon:   element.Lon,
			year:  year,
			color: normalizeColor(element.Tags["colour"]),
		}
	}

	stations := make([]importedStation, 0, len(unique))
	for _, station := range unique {
		stations = append(stations, station)
	}
	sort.Slice(stations, func(i, j int) bool {
		if stations[i].color != stations[j].color {
			return stations[i].color < stations[j].color
		}
		if stations[i].lon != stations[j].lon {
			return stations[i].lon < stations[j].lon
		}
		if stations[i].lat != stations[j].lat {
			return stations[i].lat < stations[j].lat
		}
		return stations[i].osmID < stations[j].osmID
	})
	if len(stations) == 0 {
		return nil, errors.New("в выгрузке не найдены узлы станций метро с названием")
	}
	return stations, nil
}

func buildSchema(stations []importedStation, width, height float64) outputSchema {
	minLat, maxLat := stations[0].lat, stations[0].lat
	minLon, maxLon := stations[0].lon, stations[0].lon
	for _, station := range stations[1:] {
		minLat = min(minLat, station.lat)
		maxLat = max(maxLat, station.lat)
		minLon = min(minLon, station.lon)
		maxLon = max(maxLon, station.lon)
	}

	latRange := maxLat - minLat
	lonRange := maxLon - minLon
	if latRange == 0 {
		latRange = 1
	}
	if lonRange == 0 {
		lonRange = 1
	}

	out := outputSchema{
		Stations: make([]outputStation, 0, len(stations)),
		Lines:    make([]outputLine, 0, len(lineColors)),
	}
	lineIndex := make(map[string]int)
	for index, line := range lineColors {
		lineIndex[line.color] = index
	}
	lineStations := make([][]int, len(lineColors))

	for index, station := range stations {
		x := mapMargin + (station.lon-minLon)/lonRange*(width-2*mapMargin)
		y := mapMargin + (maxLat-station.lat)/latRange*(height-2*mapMargin)
		stationID := index + 1
		out.Stations = append(out.Stations, outputStation{
			ID: stationID, Name: station.name,
			X: int(x + 0.5), Y: int(y + 0.5), Year: station.year,
		})
		if line, ok := lineIndex[station.color]; ok {
			lineStations[line] = append(lineStations[line], stationID)
		}
	}

	for index, line := range lineColors {
		if len(lineStations[index]) == 0 {
			continue
		}
		out.Lines = append(out.Lines, outputLine{
			ID: len(out.Lines) + 1, Name: line.name, Color: line.color, Stations: lineStations[index],
		})
	}
	return out
}

func parseYear(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil || year < 1800 || year > time.Now().Year()+10 {
		return 0
	}
	return year
}

func normalizeColor(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	colors := map[string]string{
		"red": "#e53935", "#ff0000": "#e53935", "#e31e24": "#e53935",
		"blue": "#1976d2", "#0000ff": "#1976d2", "#0078bf": "#1976d2",
		"green": "#2e9d50", "#008000": "#2e9d50", "#009a49": "#2e9d50",
		"orange": "#f28c28", "#ffa500": "#f28c28", "#f58220": "#f28c28",
		"purple": "#8e44ad", "#800080": "#8e44ad", "#702082": "#8e44ad",
		"brown": "#795548", "#a0522d": "#795548", "#8d5b2d": "#795548",
	}
	return colors[value]
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Ошибка:", err)
	os.Exit(1)
}
