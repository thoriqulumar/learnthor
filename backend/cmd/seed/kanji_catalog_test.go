package main

import "testing"

func TestBundledKanjiCatalogIsCompleteAndImportIsRepeatable(t *testing.T) {
	catalog, err := loadKanjiCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Entries) != 640 {
		t.Fatalf("expected 640 N5-N3 kanji, got %d", len(catalog.Entries))
	}
	counts := map[string]int{}
	for _, entry := range catalog.Entries {
		counts[entry.JLPTLevel]++
	}
	if counts["N5"] != 84 || counts["N4"] != 169 || counts["N3"] != 387 {
		t.Fatalf("unexpected level counts: %#v", counts)
	}

	database := seedTestDB(t)
	if err := importKanjiCatalog(database, catalog); err != nil {
		t.Fatal(err)
	}
	if err := importKanjiCatalog(database, catalog); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.Table("kanji_entries").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 640 {
		t.Fatalf("repeat import created duplicates: %d", count)
	}
}
