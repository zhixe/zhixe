package main

import "testing"

func TestCalculateLanguagesAggregatesAndSorts(t *testing.T) {
	repositories := []repository{
		{
			Languages: languageConnection{
				Edges: []languageEdge{
					{Size: 80, Node: struct {
						Name  string "json:\"name\""
						Color string "json:\"color\""
					}{Name: "Go", Color: "#00ADD8"}},
					{Size: 20, Node: struct {
						Name  string "json:\"name\""
						Color string "json:\"color\""
					}{Name: "Python", Color: ""}},
				},
			},
		},
		{
			Languages: languageConnection{
				Edges: []languageEdge{
					{Size: 20, Node: struct {
						Name  string "json:\"name\""
						Color string "json:\"color\""
					}{Name: "Go", Color: "#00ADD8"}},
					{Size: 50, Node: struct {
						Name  string "json:\"name\""
						Color string "json:\"color\""
					}{Name: "Rust", Color: "#DEA584"}},
					{Size: 5, Node: struct {
						Name  string "json:\"name\""
						Color string "json:\"color\""
					}{Name: "", Color: "#000000"}},
				},
			},
		},
	}

	languages := calculateLanguages(repositories)
	if len(languages) != 3 {
		t.Fatalf("expected 3 languages, got %d", len(languages))
	}

	if languages[0].Name != "Go" || languages[0].Size != 100 {
		t.Fatalf("expected first language to be Go(100), got %s(%d)", languages[0].Name, languages[0].Size)
	}

	if languages[1].Name != "Rust" || languages[1].Size != 50 {
		t.Fatalf("expected second language to be Rust(50), got %s(%d)", languages[1].Name, languages[1].Size)
	}

	if languages[2].Name != "Python" || languages[2].Color != archBlue {
		t.Fatalf("expected Python to default to archBlue, got %s", languages[2].Color)
	}
}

func TestTotalStarsAndForks(t *testing.T) {
	repositories := []repository{
		{StargazerCount: 2, ForkCount: 1},
		{StargazerCount: 5, ForkCount: 3},
		{StargazerCount: 0, ForkCount: 2},
	}

	if got := totalStars(repositories); got != 7 {
		t.Fatalf("expected 7 stars, got %d", got)
	}

	if got := totalForks(repositories); got != 6 {
		t.Fatalf("expected 6 forks, got %d", got)
	}
}
