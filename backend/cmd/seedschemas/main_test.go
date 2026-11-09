package main

import (
	"encoding/json"
	"github.com/robozmey/podzemka/backend/internal/domain"
	"testing"
)

func TestSeedSchemas(t *testing.T) {
	for slug, counts := range map[string][2]int{"saint-petersburg": {74, 6}, "moscow": {8, 2}, "krasnoyarsk": {8, 2}} {
		schema := loadSchema(domain.City{Slug: slug})
		content := domain.MetroContent{Stations: schema.Stations, Lines: schema.Lines}
		if err := content.Validate(); err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if len(content.Stations) != counts[0] || len(content.Lines) != counts[1] {
			t.Fatalf("%s: unexpected counts", slug)
		}
		data, err := json.Marshal(content)
		if err != nil {
			t.Fatal(err)
		}
		var decoded domain.MetroContent
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if slug == "saint-petersburg" && decoded.Stations[0].Year != 1982 {
			t.Fatal("opening year lost")
		}
	}
}
