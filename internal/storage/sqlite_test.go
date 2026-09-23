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

func TestSessionSummaryAndRecentContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_session.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer s.Close()

	// 1. Save session summary
	summary := core.SessionSummary{
		Goal:          "Implementar optimizaciones de contexto",
		Accomplished:  "Creadas herramientas cogni_context y cogni_session_summary",
		Discoveries:   "FTS5 provee búsqueda semántica instantánea",
		NextSteps:     "Actualizar documentación y skills",
		RelevantFiles: "internal/mcp/server.go, internal/storage/sqlite.go",
	}

	savedSession, err := s.SaveSessionSummary("viasera", "session/latest", summary, "context,compaction")
	if err != nil {
		t.Fatalf("SaveSessionSummary failed: %v", err)
	}
	if savedSession.Category != "session" {
		t.Errorf("Expected category 'session', got %s", savedSession.Category)
	}

	// 2. Also save an architectural decision
	archMem := &core.Memory{
		ProjectName:      "viasera",
		Category:         "architecture",
		Title:            "Decisión: Protocolo en 2 Fases",
		TopicKey:         "arch/retrieval/protocol",
		SummarySignature: "What: Búsqueda compacta y luego hidratación | Why: Ahorro de tokens",
		Tags:             "viasera,retrieval,tokens",
	}
	_, err = s.SaveMemory(archMem)
	if err != nil {
		t.Fatalf("SaveMemory failed: %v", err)
	}

	// 3. Test GetRecentContext
	contextMems, err := s.GetRecentContext("viasera", 5)
	if err != nil {
		t.Fatalf("GetRecentContext failed: %v", err)
	}
	if len(contextMems) < 2 {
		t.Fatalf("Expected at least 2 context memories, got %d", len(contextMems))
	}
	if contextMems[0].Category != "session" {
		t.Errorf("Expected session summary first in recent context, got category: %s", contextMems[0].Category)
	}
}

func TestCascadeBM25SearchAndFallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-cascade-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_cascade.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer s.Close()

	// 1. Create a memory in project-alpha
	memAlpha := &core.Memory{
		ProjectName:      "project-alpha",
		Category:         "architecture",
		Title:            "iOS Modal Stacking Fix and Screen Navigation",
		TopicKey:         "arch/nav/modal-stacking",
		SummarySignature: "Trigger: Modal crash on iOS | Invariant: RCTModalHostViewController cannot stack | Recipe: Use Stack.Screen",
		Tags:             "ios,modal,navigation,uikit,react-native",
	}
	saved, err := s.SaveMemory(memAlpha)
	if err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	// 2. Test Multi-Token query with extra noise words that would break strict AND
	// Notice: "crash", "failure", "error", "react" are mixed with "modal" and "ios"
	noisyQuery := "ios modal stacking navigation crash failure error react"
	resultsNoisy, err := s.SearchMemories("project-alpha", noisyQuery, "architecture", 5)
	if err != nil {
		t.Fatalf("Search with noisy query failed: %v", err)
	}
	if len(resultsNoisy) == 0 {
		t.Fatalf("Expected cascade BM25 to find the document despite extra query words, got 0 results")
	}
	if resultsNoisy[0].ID != saved.ID {
		t.Errorf("Expected ID %d, got %d", saved.ID, resultsNoisy[0].ID)
	}

	// 3. Test Cross-Project Fallback:
	// Searching from project-beta (which has no local records) for "modal stacking"
	resultsCross, err := s.SearchMemories("project-beta", "modal stacking ios", "architecture", 5)
	if err != nil {
		t.Fatalf("Cross-project search failed: %v", err)
	}
	if len(resultsCross) == 0 {
		t.Fatalf("Expected cross-project fallback to find the document from project-alpha, got 0 results")
	}
	if resultsCross[0].ID != saved.ID {
		t.Errorf("Expected cross-project result ID %d, got %d", saved.ID, resultsCross[0].ID)
	}

	// 4. Test Category Softening Fallback:
	// Document was saved as "architecture", but agent searches with category "bugfix"
	resultsCategory, err := s.SearchMemories("project-alpha", "modal stacking ios", "bugfix", 5)
	if err != nil {
		t.Fatalf("Category softening search failed: %v", err)
	}
	if len(resultsCategory) == 0 {
		t.Fatalf("Expected category softening fallback to find the document despite mismatched category, got 0 results")
	}
	if resultsCategory[0].ID != saved.ID {
		t.Errorf("Expected category softened result ID %d, got %d", saved.ID, resultsCategory[0].ID)
	}
}

func TestStorageOptimize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-optimize-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_optimize.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer s.Close()

	// Insert several memories
	for i := 1; i <= 10; i++ {
		_, err := s.SaveMemory(&core.Memory{
			ProjectName:      "test-proj",
			Category:         "architecture",
			Title:            "Architecture Note",
			TopicKey:         "arch/note",
			SummarySignature: "Signature content to fill SQLite pages with data for test purposes.",
			Tags:             "arch,testing,optimization",
		})
		if err != nil {
			t.Fatalf("Failed to save memory: %v", err)
		}
	}

	optStats, err := s.Optimize()
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}
	if optStats.TotalRows != 1 { // Upsert with same topic_key keeps 1 row
		t.Errorf("Expected 1 total row, got %d", optStats.TotalRows)
	}
	if optStats.BytesAfter <= 0 {
		t.Errorf("Expected positive BytesAfter, got %d", optStats.BytesAfter)
	}
}
