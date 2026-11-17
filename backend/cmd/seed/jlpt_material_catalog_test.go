package main

import (
	"testing"

	"github.com/learnthor/backend/models"
)

func TestBundledJLPTMaterialCatalog(t *testing.T) {
	catalog, err := loadJLPTMaterialCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Lessons) != 56 {
		t.Fatalf("expected 56 lessons, got %d", len(catalog.Lessons))
	}
	counts := map[string]int{}
	for _, lesson := range catalog.Lessons {
		counts[lesson.Level]++
	}
	for level, expected := range map[string]int{"N5": 22, "N4": 14, "N3": 20} {
		if counts[level] != expected {
			t.Fatalf("expected %d %s lessons, got %d", expected, level, counts[level])
		}
	}
}

func TestJLPTMaterialImportIsRepeatable(t *testing.T) {
	database := seedTestDB(t)
	catalog, err := loadJLPTMaterialCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if err := importJLPTMaterialCatalog(database, catalog); err != nil {
		t.Fatal(err)
	}
	if err := importJLPTMaterialCatalog(database, catalog); err != nil {
		t.Fatalf("repeat import: %v", err)
	}
	var lessonCount, questionCount int64
	if err := database.Model(&models.JLPTLesson{}).Count(&lessonCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&models.JLPTQuestion{}).Count(&questionCount).Error; err != nil {
		t.Fatal(err)
	}
	if lessonCount != 56 || questionCount != 168 {
		t.Fatalf("unexpected counts after repeat import: lessons=%d questions=%d", lessonCount, questionCount)
	}
}
