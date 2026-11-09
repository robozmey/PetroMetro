package domain

import (
	"fmt"
	"regexp"
	"strings"
)

type MetroContent struct {
	Stations []Station   `json:"stations"`
	Lines    []MetroLine `json:"lines"`
}

var validColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s MetroContent) Validate() error {
	if s.Stations == nil || s.Lines == nil {
		return fmt.Errorf("stations and lines must be arrays")
	}
	stations := make(map[int]bool)
	for _, station := range s.Stations {
		if station.ID <= 0 || stations[station.ID] {
			return fmt.Errorf("invalid or duplicate station id: %d", station.ID)
		}
		if strings.TrimSpace(station.Name) == "" {
			return fmt.Errorf("station %d has empty name", station.ID)
		}
		if station.Year != 0 && (station.Year < 1863 || station.Year > 2100) {
			return fmt.Errorf("station %d has invalid year: %d", station.ID, station.Year)
		}

		stations[station.ID] = true
	}
	lines := make(map[int]bool)
	for _, line := range s.Lines {
		if line.ID <= 0 || lines[line.ID] {
			return fmt.Errorf("invalid or duplicate line id: %d", line.ID)
		}
		if strings.TrimSpace(line.Name) == "" {
			return fmt.Errorf("line %d has empty name", line.ID)
		}

		if !validColor.MatchString(line.Color) {
			return fmt.Errorf("line %d has invalid color: %s", line.ID, line.Color)
		}

		lines[line.ID] = true

		seen := make(map[int]bool)
		if line.Stations == nil {
			return fmt.Errorf("line %d stations must be an array", line.ID)
		}
		for _, id := range line.Stations {
			if !stations[id] {
				return fmt.Errorf("line %d references missing station %d", line.ID, id)
			}
			if seen[id] {
				return fmt.Errorf("line %d has duplicate station %d", line.ID, id)
			}
			seen[id] = true
		}
	}
	return nil
}
