package main

import (
	"strings"
	"testing"

	"github.com/robozmey/podzemka/backend/internal/overpass"
)

func TestImporterUsesSharedOverpassParser(t *testing.T) {
	input := `{"elements":[{"type":"node","id":1,"lat":59.9,"lon":30.3,"tags":{"station":"subway","name":"Station A","colour":"blue"}},{"type":"node","id":2,"lat":59.91,"lon":30.31,"tags":{"station":"subway","name":"Station B","colour":"blue"}}]}`
	schema, err := overpass.Parse(strings.NewReader(input), overpass.MapWidth, overpass.MapHeight)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(schema.Stations) != 2 || len(schema.Lines) != 1 {
		t.Fatalf("parsed stations=%d lines=%d; want 2 and 1", len(schema.Stations), len(schema.Lines))
	}
}
