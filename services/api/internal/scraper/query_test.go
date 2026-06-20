package scraper

import "testing"

func TestSearchQueries(t *testing.T) {
	qs := SearchQueries("加勒比海盗5：死无对证")
	if len(qs) < 2 {
		t.Fatalf("expected multiple queries, got %v", qs)
	}
	if qs[0] != "加勒比海盗5：死无对证" {
		t.Fatalf("unexpected first query: %q", qs[0])
	}
	foundShort := false
	for _, q := range qs {
		if q == "加勒比海盗5" {
			foundShort = true
		}
	}
	if !foundShort {
		t.Fatalf("expected shortened query, got %v", qs)
	}
}

func TestSearchQueries_StripsBracketTags(t *testing.T) {
	qs := SearchQueries("葫芦小金刚[4KHDR.CN]Calabash Brothers")
	if len(qs) == 0 || qs[0] != "葫芦小金刚Calabash Brothers" {
		t.Fatalf("got %v", qs)
	}
}

func TestSearchQueries_NormalizesSeriesTitle(t *testing.T) {
	qs := SearchQueries("唐丨朝诡事录之西行(2024)")
	found := map[string]bool{}
	for _, q := range qs {
		found[q] = true
	}
	if !found["唐朝诡事录之西行"] {
		t.Fatalf("missing cleaned title, got %v", qs)
	}
	if !found["唐朝诡事录"] {
		t.Fatalf("missing parent title, got %v", qs)
	}
}
