package overpass

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
)

const (
	MapWidth  = 700.0
	MapHeight = 420.0
	mapMargin = 30.0
)

type response struct {
	Elements []element `json:"elements"`
}

type rawElement = element

type element struct {
	Type string            `json:"type"`
	ID   int64             `json:"id"`
	Lat  float64           `json:"lat"`
	Lon  float64           `json:"lon"`
	Tags map[string]string `json:"tags"`
}

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

// Parse converts a raw Overpass JSON export into a metro schema content object.
func Parse(input io.Reader, width, height float64) (domain.MetroContent, error) {
	if width <= 2*mapMargin || height <= 2*mapMargin {
		return domain.MetroContent{}, errors.New("ширина и высота должны быть больше двойного отступа")
	}
	stations, err := parseStations(input)
	if err != nil {
		return domain.MetroContent{}, err
	}
	return buildSchema(stations, width, height), nil
}

func parseStations(input io.Reader) ([]importedStation, error) {
	payload, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("прочитать Overpass JSON: %w", err)
	}

	var data response
	if err := json.Unmarshal(payload, &data); err != nil {
		// Some exports contain comma-separated nodes without the enclosing
		// {"elements": [...]} object and may have trailing commas.
		cleanPayload := removeTrailingCommas(bytes.TrimSpace(payload))
		var rawElements []rawElement
		wrapped := append(append([]byte{'['}, cleanPayload...), ']')
		if arrayErr := json.Unmarshal(wrapped, &rawElements); arrayErr != nil {
			return nil, fmt.Errorf("разобрать Overpass JSON: %w", err)
		}
		data.Elements = rawElements
	}

	unique := make(map[int64]importedStation)
	for _, item := range data.Elements {
		if item.Type != "node" || item.ID == 0 || item.Lat == 0 || item.Lon == 0 {
			continue
		}
		if item.Tags["station"] != "subway" && item.Tags["subway"] != "yes" {
			continue
		}
		name := strings.TrimSpace(item.Tags["name:ru"])
		if name == "" {
			name = strings.TrimSpace(item.Tags["name"])
		}
		if name == "" {
			continue
		}
		unique[item.ID] = importedStation{
			osmID: item.ID,
			name:  name,
			lat:   item.Lat,
			lon:   item.Lon,
			year:  parseYear(item.Tags["start_date"]),
			color: normalizeColor(item.Tags["colour"]),
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
		return stations[i].osmID < stations[j].osmID
	})
	if len(stations) == 0 {
		return nil, errors.New("в выгрузке не найдены узлы станций метро с названием")
	}
	return stations, nil
}

func removeTrailingCommas(payload []byte) []byte {
	cleaned := make([]byte, 0, len(payload))
	inString := false
	escaped := false
	for i, char := range payload {
		if inString {
			cleaned = append(cleaned, char)
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' {
			inString = true
			cleaned = append(cleaned, char)
			continue
		}
		if char == ',' {
			j := i + 1
			for j < len(payload) && (payload[j] == ' ' || payload[j] == '\n' || payload[j] == '\r' || payload[j] == '\t') {
				j++
			}
			if j < len(payload) && (payload[j] == '}' || payload[j] == ']') {
				continue
			}
		}
		cleaned = append(cleaned, char)
	}
	return cleaned
}

func buildSchema(stations []importedStation, width, height float64) domain.MetroContent {
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

	out := domain.MetroContent{
		Stations: make([]domain.Station, 0, len(stations)),
		Lines:    make([]domain.MetroLine, 0, len(lineColors)),
	}
	lineIndex := make(map[string]int)
	for index, line := range lineColors {
		lineIndex[line.color] = index
	}
	lineStations := make([][]importedStation, len(lineColors))
	stationIDsByOSMID := make(map[int64]int, len(stations))
	for index, station := range stations {
		x := mapMargin + (station.lon-minLon)/lonRange*(width-2*mapMargin)
		y := mapMargin + (maxLat-station.lat)/latRange*(height-2*mapMargin)
		stationID := index + 1
		out.Stations = append(out.Stations, domain.Station{
			ID: stationID, Name: station.name,
			X: int(x + 0.5), Y: int(y + 0.5), Year: station.year,
		})
		stationIDsByOSMID[station.osmID] = stationID
		if line, ok := lineIndex[station.color]; ok {
			lineStations[line] = append(lineStations[line], station)
		}
	}
	for index, line := range lineColors {
		orderedStations := orderLineStations(lineStations[index])
		if len(orderedStations) == 0 {
			continue
		}
		stationIDs := make([]int, 0, len(orderedStations))
		for _, station := range orderedStations {
			stationIDs = append(stationIDs, stationIDsByOSMID[station.osmID])
		}
		out.Lines = append(out.Lines, domain.MetroLine{
			ID: len(out.Lines) + 1, Name: line.name, Color: line.color, Stations: stationIDs,
		})
	}
	return out
}

func orderLineStations(stations []importedStation) []importedStation {
	if len(stations) < 2 {
		return append([]importedStation(nil), stations...)
	}
	remaining := append([]importedStation(nil), stations...)
	sort.Slice(remaining, func(i, j int) bool {
		if remaining[i].lon != remaining[j].lon {
			return remaining[i].lon < remaining[j].lon
		}
		if remaining[i].lat != remaining[j].lat {
			return remaining[i].lat < remaining[j].lat
		}
		return remaining[i].osmID < remaining[j].osmID
	})
	ordered := make([]importedStation, 0, len(remaining))
	current := remaining[0]
	ordered = append(ordered, current)
	remaining = remaining[1:]
	for len(remaining) > 0 {
		nearestIndex := 0
		nearestDistance := squaredGeoDistance(current, remaining[0])
		for i := 1; i < len(remaining); i++ {
			distance := squaredGeoDistance(current, remaining[i])
			if distance < nearestDistance || (distance == nearestDistance && remaining[i].osmID < remaining[nearestIndex].osmID) {
				nearestIndex = i
				nearestDistance = distance
			}
		}
		current = remaining[nearestIndex]
		ordered = append(ordered, current)
		remaining = append(remaining[:nearestIndex], remaining[nearestIndex+1:]...)
	}
	return ordered
}

func squaredGeoDistance(a, b importedStation) float64 {
	meanLat := (a.lat + b.lat) / 2 * math.Pi / 180
	dLon := (a.lon - b.lon) * math.Cos(meanLat)
	dLat := a.lat - b.lat
	return dLon*dLon + dLat*dLat
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
