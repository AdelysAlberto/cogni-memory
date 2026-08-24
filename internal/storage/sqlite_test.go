package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AdelysAlberto/cogni/internal/core"
)

func TestStorageCRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_memory.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer s.Close()

	// 1. Save
	mem := &core.Memory{
		ProjectName:      "test-app",
		Category:         "testing",
		Title:            "Unit Testing Core",
		TopicKey:         "arch/core/testing",
		SummarySignature: "Test signature for sqlite CRUD functionality.",
		Tags:             "test-app,unit-test,sqlite",
	}

	saved, err := s.SaveMemory(mem)
	if err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}
	if saved.ID <= 0 {
		t.Errorf("Expected positive ID, got %d", saved.ID)
	}
	if saved.TopicKey != "arch/core/testing" {
		t.Errorf("Expected TopicKey arch/core/testing, got %s", saved.TopicKey)
	}

	// 1b. Test Upsert with same TopicKey
	upsertMem := &core.Memory{
		ProjectName:      "test-app",
		Category:         "testing",
		Title:            "Unit Testing Core Upserted",
		TopicKey:         "arch/core/testing",
		SummarySignature: "Upserted signature content.",
		Tags:             "test-app,unit-test,sqlite,upsert",
	}
	upserted, err := s.SaveMemory(upsertMem)
	if err != nil {
		t.Fatalf("Failed to upsert memory: %v", err)
	}
	if upserted.ID != saved.ID {
		t.Errorf("Expected same ID on upsert (%d), got %d", saved.ID, upserted.ID)
	}
	if upserted.Title != "Unit Testing Core Upserted" {
		t.Errorf("Expected title 'Unit Testing Core Upserted', got %s", upserted.Title)
	}

	// 2. Search
	results, err := s.SearchMemories("test-app", "CRUD", "", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		// Try searching with "Upserted"
		results, err = s.SearchMemories("test-app", "Upserted", "", 10)
		if err != nil || len(results) == 0 {
			t.Errorf("Expected at least 1 search result, got 0")
		}
	}

	// 3. Update
	updated, err := s.UpdateMemory(saved.ID, "Unit Testing Core V2", "Updated signature", "testing", "test-app,sqlite", "arch/core/testing")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Unit Testing Core V2" {
		t.Errorf("Expected updated title, got %s", updated.Title)
	}

	// 4. Stats
	stats, err := s.GetStats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TotalMemories != 1 {
		t.Errorf("Expected 1 memory in stats, got %d", stats.TotalMemories)
	}

	// 5. Delete
	deleted, err := s.DeleteMemory(saved.ID)
	if err != nil || !deleted {
		t.Fatalf("Delete failed: %v", err)
	}

	// 6. Verify Deleted
	afterDel, err := s.GetMemoryByID(saved.ID)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if afterDel != nil {
		t.Errorf("Expected nil memory after deletion, got %+v", afterDel)
	}
}

func TestFTSTagsAndStemSearch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-fts-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_fts.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer s.Close()

	// Save memory with Spanish Title and English tag / topic_key
	mem := &core.Memory{
		ProjectName:      "viasera",
		Category:         "config",
		Title:            "Utilidad de Configuración Dinámica con Caché Redis y Fallback (config.util)",
		TopicKey:         "config.util",
		SummarySignature: "What: Creadas funciones getSystemConfig y setSystemConfig | Why: Redis cache | Where: src/config.ts",
		Tags:             "viasera,utils,config,redis,cache,fallback",
	}

	saved, err := s.SaveMemory(mem)
	if err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	// 1. Search by exact English tag "utils" -> should match via FTS live trigger and tag index!
	resUtils, err := s.SearchMemories("viasera", "utils", "", 10)
	if err != nil {
		t.Fatalf("Search by 'utils' failed: %v", err)
	}
	if len(resUtils) == 0 {
		t.Errorf("Expected to find memory by tag 'utils', got 0 results")
	} else if resUtils[0].ID != saved.ID {
		t.Errorf("Expected memory ID %d, got %d", saved.ID, resUtils[0].ID)
	}

	// 2. Search by "utils" stem matching "Utilidad" in Spanish Title
	resUtil, err := s.SearchMemories("viasera", "util", "", 10)
	if err != nil || len(resUtil) == 0 {
		t.Errorf("Expected to find memory by stem 'util', got 0 results")
	}

	// 3. Search by topic_key "config.util"
	resTopic, err := s.SearchMemories("viasera", "config.util", "", 10)
	if err != nil || len(resTopic) == 0 {
		t.Errorf("Expected to find memory by topic_key 'config.util', got 0 results")
	}
}
