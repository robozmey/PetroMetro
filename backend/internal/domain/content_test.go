package domain

import "testing"

func TestMetroContentValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*MetroContent)
		want   string
	}{
		{"valid", func(s *MetroContent) {}, ""},
		{"empty arrays", func(s *MetroContent) { s.Stations = []Station{}; s.Lines = []MetroLine{} }, ""},
		{"missing stations", func(s *MetroContent) { s.Stations = nil }, "stations and lines must be arrays"},
		{"missing lines", func(s *MetroContent) { s.Lines = nil }, "stations and lines must be arrays"},
		{"zero station id", func(s *MetroContent) { s.Stations[0].ID = 0 }, "invalid or duplicate station id: 0"},
		{"negative station id", func(s *MetroContent) { s.Stations[0].ID = -1 }, "invalid or duplicate station id: -1"},
		{"duplicate station", func(s *MetroContent) { s.Stations = append(s.Stations, s.Stations[0]) }, "invalid or duplicate station id: 1"},
		{"empty station name", func(s *MetroContent) { s.Stations[0].Name = "" }, "station 1 has empty name"},
		{"blank station name", func(s *MetroContent) { s.Stations[0].Name = " \t" }, "station 1 has empty name"},
		{"unknown year", func(s *MetroContent) { s.Stations[0].Year = 0 }, ""},
		{"minimum year", func(s *MetroContent) { s.Stations[0].Year = 1863 }, ""},
		{"maximum year", func(s *MetroContent) { s.Stations[0].Year = 2100 }, ""},
		{"year too early", func(s *MetroContent) { s.Stations[0].Year = 1862 }, "station 1 has invalid year: 1862"},
		{"year too late", func(s *MetroContent) { s.Stations[0].Year = 2101 }, "station 1 has invalid year: 2101"},
		{"zero line id", func(s *MetroContent) { s.Lines[0].ID = 0 }, "invalid or duplicate line id: 0"},
		{"negative line id", func(s *MetroContent) { s.Lines[0].ID = -1 }, "invalid or duplicate line id: -1"},
		{"duplicate line", func(s *MetroContent) { s.Lines = append(s.Lines, s.Lines[0]) }, "invalid or duplicate line id: 1"},
		{"empty line name", func(s *MetroContent) { s.Lines[0].Name = "" }, "line 1 has empty name"},
		{"blank line name", func(s *MetroContent) { s.Lines[0].Name = " \t" }, "line 1 has empty name"},
		{"mixed case color", func(s *MetroContent) { s.Lines[0].Color = "#aBcDeF" }, ""},
		{"short color", func(s *MetroContent) { s.Lines[0].Color = "#abc" }, "line 1 has invalid color: #abc"},
		{"nonhex color", func(s *MetroContent) { s.Lines[0].Color = "#gg0000" }, "line 1 has invalid color: #gg0000"},
		{"missing color prefix", func(s *MetroContent) { s.Lines[0].Color = "112233" }, "line 1 has invalid color: 112233"},
		{"missing line stations", func(s *MetroContent) { s.Lines[0].Stations = nil }, "line 1 stations must be an array"},
		{"empty line stations", func(s *MetroContent) { s.Lines[0].Stations = []int{} }, ""},
		{"missing station reference", func(s *MetroContent) { s.Lines[0].Stations = []int{42} }, "line 1 references missing station 42"},
		{"duplicate station in line", func(s *MetroContent) { s.Lines[0].Stations = []int{1, 1} }, "line 1 has duplicate station 1"},
		{"station shared by lines", func(s *MetroContent) { line := s.Lines[0]; line.ID = 2; s.Lines = append(s.Lines, line) }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := MetroContent{
				Stations: []Station{{ID: 1, Name: "Station", Year: 1982}},
				Lines:    []MetroLine{{ID: 1, Name: "Line", Color: "#123abc", Stations: []int{1}}},
			}
			tt.change(&content)
			err := content.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q", tt.want)
			}
			if err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}
