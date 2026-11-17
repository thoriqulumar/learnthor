package services

import (
	"testing"

	"github.com/learnthor/backend/models"
)

func TestKanjiWordMergesTranslationsForSameReading(t *testing.T) {
	database := newTestDB(t)
	fixture := seedCurriculum(t, database, 1)
	word, translation := "明日", "esok, besok"
	items := []models.VocabItem{
		{ModuleID: fixture.Module.ID, Kanji: &word, Kana: "あした", EnglishMeaning: "Tomorrow", ExampleSentence: "あした学校に行きます。"},
		{ModuleID: fixture.Module.ID, Kanji: &word, Kana: "あした", EnglishMeaning: "tomorrow; near future", IndonesianMeaning: &translation, Romaji: "ashita"},
		{ModuleID: fixture.Module.ID, Kanji: &word, Kana: "あす", EnglishMeaning: "tomorrow", Romaji: "asu"},
	}
	if err := database.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	results, err := NewKanjiService(database).Word(word)
	if err != nil || len(results) != 2 {
		t.Fatalf("unexpected readings: %#v, %v", results, err)
	}
	entry := results[0]
	if entry.IndonesianMeaning == nil || *entry.IndonesianMeaning != translation || entry.EnglishMeaning != "Tomorrow; near future" || entry.Romaji != "ashita" || entry.ExampleSentence == "" {
		t.Fatalf("word details were not merged: %#v", entry)
	}
}

func TestKanjiServiceListsAndSearchesBilingualCatalog(t *testing.T) {
	database := newTestDB(t)
	entries := []models.KanjiEntry{
		{SeedKey: "kanji:n5-day", Character: "日", JLPTLevel: "N5", SortOrder: 1, StrokeCount: 4, OnReadings: "ニチ|ジツ", KunReadings: "ひ|-び", EnglishMeaning: "day, sun", IndonesianMeaning: "hari, matahari", ExampleWords: "日本|今日", StrokeSVG: "/kanji/065e5.svg"},
		{SeedKey: "kanji:n3-cat", Character: "猫", JLPTLevel: "N3", SortOrder: 1, StrokeCount: 11, OnReadings: "ビョウ", KunReadings: "ねこ", EnglishMeaning: "cat", IndonesianMeaning: "kucing", StrokeSVG: "/kanji/0732b.svg"},
	}
	if err := database.Create(&entries).Error; err != nil {
		t.Fatal(err)
	}

	service := NewKanjiService(database)
	all, err := service.List("", "")
	if err != nil || len(all) != 2 || all[0].Character != "日" {
		t.Fatalf("unexpected list: %#v, %v", all, err)
	}
	results, err := service.List("N3", "kucing")
	if err != nil || len(results) != 1 || results[0].Character != "猫" {
		t.Fatalf("unexpected bilingual search: %#v, %v", results, err)
	}
	if len(results[0].KunReadings) != 1 || results[0].KunReadings[0] != "ねこ" {
		t.Fatalf("readings were not decoded: %#v", results[0].KunReadings)
	}
}
