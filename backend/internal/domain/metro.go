package domain

type City struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Station struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	CityID int    `json:"-"`
	Year   int    `json:"year,omitempty"`
}

type MetroLine struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Stations []int  `json:"stations"`
}

type MetroSchema struct {
	City     City        `json:"city"`
	Stations []Station   `json:"stations"`
	Lines    []MetroLine `json:"lines"`
}
