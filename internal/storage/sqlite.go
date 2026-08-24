package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db     *sql.DB
	dbPath string
	source string
}

// New creates and initializes a SQLite storage engine
func New(dbPath string) (*Storage, error) {
	return NewWithSource(dbPath, "")
}

// NewWithSource creates and initializes a SQLite storage engine with explicit source tag
func NewWithSource(dbPath string, source string) (*Storage, error) {
	// Auto migrate legacy database if applicable
	migrateLegacyDatabase(dbPath)

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database at %s: %w", dbPath, err)
	}

	// Optimize pragmas for fast concurrent reads and WAL mode
	_, _ = db.Exec("PRAGMA journal_mode = WAL;")
	_, _ = db.Exec("PRAGMA synchronous = NORMAL;")
	_, _ = db.Exec("PRAGMA busy_timeout = 5000;")

	s := &Storage{db: db, dbPath: dbPath, source: source}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SetSource updates the source label
func (s *Storage) SetSource(src string) {
	s.source = src
}

// Source returns the source label (local or global)
func (s *Storage) Source() string {
	return s.source
}

// DBPath returns current database file path
func (s *Storage) DBPath() string {
	return s.dbPath
}

func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_name TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'general',
		title TEXT NOT NULL,
		topic_key TEXT NOT NULL DEFAULT '',
		summary_signature TEXT NOT NULL,
		tags TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Check existing columns via PRAGMA
	cols := make(map[string]bool)
	rows, err := s.db.Query("PRAGMA table_info(agent_memories)")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue any
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
				cols[strings.ToLower(name)] = true
			}
		}
		rows.Close()
	}

	if !cols["topic_key"] {
		_, _ = s.db.Exec("ALTER TABLE agent_memories ADD COLUMN topic_key TEXT NOT NULL DEFAULT ''")
	}
	if !cols["created_at"] {
		_, _ = s.db.Exec("ALTER TABLE agent_memories ADD COLUMN created_at DATETIME DEFAULT ''")
	}
	if !cols["updated_at"] {
		_, _ = s.db.Exec("ALTER TABLE agent_memories ADD COLUMN updated_at DATETIME DEFAULT ''")
	}

	// Safely create indexes after all columns are guaranteed to exist
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_memories_project ON agent_memories(project_name);")
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_memories_category ON agent_memories(category);")
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_memories_topic_key ON agent_memories(project_name, topic_key);")

	// Backfill created_at and updated_at
	_, _ = s.db.Exec(`
		UPDATE agent_memories 
		SET created_at = CASE 
			WHEN created_at IS NOT NULL AND created_at != '' THEN created_at
			WHEN timestamp IS NOT NULL AND timestamp != '' THEN timestamp
			ELSE datetime('now')
		END
		WHERE created_at IS NULL OR created_at = ''
	`)
	_, _ = s.db.Exec("UPDATE agent_memories SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = ''")

	// Create index on created_at safely
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_memories_created_at ON agent_memories(created_at DESC)")

	// FTS5 Setup & Triggers for live synchronization
	// Drop old FTS table/triggers if missing topic_key column or outdated schema
	_, _ = s.db.Exec("DROP TRIGGER IF EXISTS agent_memories_ai")
	_, _ = s.db.Exec("DROP TRIGGER IF EXISTS agent_memories_ad")
	_, _ = s.db.Exec("DROP TRIGGER IF EXISTS agent_memories_au")
	_, _ = s.db.Exec("DROP TABLE IF EXISTS memories_fts")

	ftsSchema := `
	CREATE VIRTUAL TABLE memories_fts USING fts5(
		title,
		topic_key,
		summary_signature,
		tags,
		content='agent_memories',
		content_rowid='id'
	);

	CREATE TRIGGER agent_memories_ai AFTER INSERT ON agent_memories BEGIN
		INSERT INTO memories_fts(rowid, title, topic_key, summary_signature, tags)
		VALUES (new.id, new.title, new.topic_key, new.summary_signature, new.tags);
	END;

	CREATE TRIGGER agent_memories_ad AFTER DELETE ON agent_memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, title, topic_key, summary_signature, tags)
		VALUES ('delete', old.id, old.title, old.topic_key, old.summary_signature, old.tags);
	END;

	CREATE TRIGGER agent_memories_au AFTER UPDATE ON agent_memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, title, topic_key, summary_signature, tags)
		VALUES ('delete', old.id, old.title, old.topic_key, old.summary_signature, old.tags);
		INSERT INTO memories_fts(rowid, title, topic_key, summary_signature, tags)
		VALUES (new.id, new.title, new.topic_key, new.summary_signature, new.tags);
	END;
	`
	_, _ = s.db.Exec(ftsSchema)

	// Rebuild FTS index from existing memories
	_, _ = s.db.Exec("INSERT OR REPLACE INTO memories_fts(rowid, title, topic_key, summary_signature, tags) SELECT id, title, topic_key, summary_signature, tags FROM agent_memories")

	return nil
}

// migrateLegacyDatabase copies ~/.agent-memory/memory.db to ~/.cogni/memory.db if cogni DB does not exist
func migrateLegacyDatabase(targetPath string) {
	if _, err := os.Stat(targetPath); err == nil {
		return // Target already exists
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Only migrate if target is ~/.cogni/memory.db
	expectedGlobal := filepath.Join(home, ".cogni", "memory.db")
	if filepath.Clean(targetPath) != filepath.Clean(expectedGlobal) {
		return
	}

	legacyPath := filepath.Join(home, ".agent-memory", "memory.db")
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return
	}

	_ = os.MkdirAll(filepath.Dir(targetPath), 0755)

	src, err := os.Open(legacyPath)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(targetPath)
	if err != nil {
		return
	}
	defer dst.Close()

	_, _ = io.Copy(dst, src)
}

// SaveMemory creates a new memory record, or updates existing record if topic_key matches
func (s *Storage) SaveMemory(m *core.Memory) (*core.Memory, error) {
	// If topic_key is provided, check for existing record in the same project to perform an upsert
	if m.TopicKey != "" {
		existing, err := s.GetMemoryByTopicKey(m.ProjectName, m.TopicKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			query := `
				UPDATE agent_memories 
				SET title = ?, summary_signature = ?, category = ?, tags = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`
			_, err = s.db.Exec(query, m.Title, m.SummarySignature, m.Category, m.Tags, existing.ID)
			if err != nil {
				return nil, fmt.Errorf("error updating memory by topic_key: %w", err)
			}
			return s.GetMemoryByID(existing.ID)
		}
	}

	query := `
		INSERT INTO agent_memories (project_name, category, title, topic_key, summary_signature, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	res, err := s.db.Exec(query, m.ProjectName, m.Category, m.Title, m.TopicKey, m.SummarySignature, m.Tags)
	if err != nil {
		return nil, fmt.Errorf("error saving memory: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	m.ID = id

	return s.GetMemoryByID(id)
}

// UpdateMemory updates fields of an existing memory
func (s *Storage) UpdateMemory(id int64, title, summary, category, tags, topicKey string) (*core.Memory, error) {
	existing, err := s.GetMemoryByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("memory with ID %d not found", id)
	}

	if title != "" {
		existing.Title = title
	}
	if summary != "" {
		existing.SummarySignature = summary
	}
	if category != "" {
		existing.Category = category
	}
	if tags != "" {
		existing.Tags = tags
	}
	if topicKey != "" {
		existing.TopicKey = topicKey
	}

	query := `
		UPDATE agent_memories 
		SET title = ?, topic_key = ?, summary_signature = ?, category = ?, tags = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err = s.db.Exec(query, existing.Title, existing.TopicKey, existing.SummarySignature, existing.Category, existing.Tags, id)
	if err != nil {
		return nil, fmt.Errorf("error updating memory: %w", err)
	}

	return s.GetMemoryByID(id)
}

// DeleteMemory deletes a memory by ID
func (s *Storage) DeleteMemory(id int64) (bool, error) {
	res, err := s.db.Exec("DELETE FROM agent_memories WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("error deleting memory: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// GetMemoryByID retrieves a single memory by ID
func (s *Storage) GetMemoryByID(id int64) (*core.Memory, error) {
	query := `
		SELECT id, project_name, category, title, topic_key, summary_signature, tags, 
		       COALESCE(created_at, CURRENT_TIMESTAMP), 
		       COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM agent_memories
		WHERE id = ?
	`
	row := s.db.QueryRow(query, id)

	var m core.Memory
	var createdAtStr, updatedAtStr string
	err := row.Scan(&m.ID, &m.ProjectName, &m.Category, &m.Title, &m.TopicKey, &m.SummarySignature, &m.Tags, &createdAtStr, &updatedAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading memory: %w", err)
	}

	m.CreatedAt = parseTime(createdAtStr)
	m.UpdatedAt = parseTime(updatedAtStr)
	m.Source = s.source
	return &m, nil
}

// GetMemoryByTopicKey retrieves a single memory by project_name and topic_key
func (s *Storage) GetMemoryByTopicKey(projectName, topicKey string) (*core.Memory, error) {
	if topicKey == "" {
		return nil, nil
	}
	query := `
		SELECT id, project_name, category, title, topic_key, summary_signature, tags, 
		       COALESCE(created_at, CURRENT_TIMESTAMP), 
		       COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM agent_memories
		WHERE (? = '' OR project_name = ? OR project_name = 'global')
		  AND topic_key = ?
		ORDER BY CASE WHEN project_name = ? THEN 0 ELSE 1 END, updated_at DESC
		LIMIT 1
	`
	row := s.db.QueryRow(query, projectName, projectName, topicKey, projectName)

	var m core.Memory
	var createdAtStr, updatedAtStr string
	err := row.Scan(&m.ID, &m.ProjectName, &m.Category, &m.Title, &m.TopicKey, &m.SummarySignature, &m.Tags, &createdAtStr, &updatedAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading memory by topic_key: %w", err)
	}

	m.CreatedAt = parseTime(createdAtStr)
	m.UpdatedAt = parseTime(updatedAtStr)
	m.Source = s.source
	return &m, nil
}

// buildFTSQuery cleans input and expands terms with stem prefixes (e.g. utils -> utils* OR util*)
func buildFTSQuery(query string) string {
	sanitized := strings.TrimSpace(query)
	if sanitized == "" {
		return ""
	}

	// Remove FTS special characters that break FTS syntax
	replacer := strings.NewReplacer(
		"\"", " ", "'", " ", "*", " ", ":", " ", "(", " ", ")", " ",
		"-", " ", "^", " ", "{", " ", "}", " ", "#", " ", ",", " ",
	)
	clean := replacer.Replace(sanitized)
	words := strings.Fields(clean)
	if len(words) == 0 {
		return ""
	}

	var terms []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		// Generate stem prefix for common plurals / English-Spanish cross-matches
		lowerW := strings.ToLower(w)
		stem := lowerW
		if strings.HasSuffix(lowerW, "utils") {
			stem = strings.TrimSuffix(lowerW, "s") // "util"
		} else if strings.HasSuffix(lowerW, "es") && len(lowerW) > 4 {
			stem = lowerW[:len(lowerW)-2]
		} else if strings.HasSuffix(lowerW, "s") && len(lowerW) > 3 {
			stem = lowerW[:len(lowerW)-1]
		}

		if stem != lowerW && len(stem) >= 3 {
			terms = append(terms, fmt.Sprintf("(%s* OR %s*)", w, stem))
		} else {
			terms = append(terms, fmt.Sprintf("%s*", w))
		}
	}
	return strings.Join(terms, " ")
}

// SearchMemories performs full-text or fuzzy search across memories
func (s *Storage) SearchMemories(projectName, query, category string, limit int) ([]core.Memory, error) {
	if limit <= 0 {
		limit = 10
	}

	var memories []core.Memory

	// 1. Try FTS5 match first if query is non-empty
	ftsArg := buildFTSQuery(query)
	if ftsArg != "" {
		ftsQuery := `
			SELECT m.id, m.project_name, m.category, m.title, m.topic_key, m.summary_signature, m.tags,
			       COALESCE(m.created_at, CURRENT_TIMESTAMP), COALESCE(m.updated_at, CURRENT_TIMESTAMP)
			FROM memories_fts f
			JOIN agent_memories m ON f.rowid = m.id
			WHERE (? = '' OR m.project_name = ? OR m.project_name = 'global')
			  AND (? = '' OR m.category = ?)
			  AND memories_fts MATCH ?
			ORDER BY rank
			LIMIT ?
		`
		rows, err := s.db.Query(ftsQuery, projectName, projectName, category, category, ftsArg, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var m core.Memory
				var cStr, uStr string
				if err := rows.Scan(&m.ID, &m.ProjectName, &m.Category, &m.Title, &m.TopicKey, &m.SummarySignature, &m.Tags, &cStr, &uStr); err == nil {
					m.CreatedAt = parseTime(cStr)
					m.UpdatedAt = parseTime(uStr)
					m.Source = s.source
					memories = append(memories, m)
				}
			}
			if len(memories) > 0 {
				return memories, nil
			}
		}
	}

	// 2. Fallback to LIKE query for multi-word / fuzzy matching
	words := strings.Fields(strings.ReplaceAll(query, "#", ""))
	if len(words) == 0 {
		return memories, nil
	}

	var whereClauses []string
	var args []interface{}

	whereClauses = append(whereClauses, "(? = '' OR project_name = ? OR project_name = 'global')")
	args = append(args, projectName, projectName)

	whereClauses = append(whereClauses, "(? = '' OR category = ?)")
	args = append(args, category, category)

	for _, word := range words {
		pattern := "%" + word + "%"
		whereClauses = append(whereClauses, "(title LIKE ? OR topic_key LIKE ? OR summary_signature LIKE ? OR tags LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}

	likeQuery := fmt.Sprintf(`
		SELECT id, project_name, category, title, topic_key, summary_signature, tags,
		       COALESCE(created_at, CURRENT_TIMESTAMP), COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM agent_memories
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ?
	`, strings.Join(whereClauses, " AND "))

	args = append(args, limit)
	rows, err := s.db.Query(likeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m core.Memory
		var cStr, uStr string
		if err := rows.Scan(&m.ID, &m.ProjectName, &m.Category, &m.Title, &m.TopicKey, &m.SummarySignature, &m.Tags, &cStr, &uStr); err != nil {
			return nil, err
		}
		m.CreatedAt = parseTime(cStr)
		m.UpdatedAt = parseTime(uStr)
		m.Source = s.source
		memories = append(memories, m)
	}

	return memories, nil
}

// ListMemories returns memories optionally filtered by project and category
func (s *Storage) ListMemories(projectName, category string, limit, offset int) ([]core.Memory, error) {
	if limit <= 0 {
		limit = 50
	}

	sqlQuery := `
		SELECT id, project_name, category, title, topic_key, summary_signature, tags,
		       COALESCE(created_at, CURRENT_TIMESTAMP), COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM agent_memories
		WHERE (? = '' OR project_name = ? OR project_name = 'global')
		  AND (? = '' OR category = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := s.db.Query(sqlQuery, projectName, projectName, category, category, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list error: %w", err)
	}
	defer rows.Close()

	var memories []core.Memory
	for rows.Next() {
		var m core.Memory
		var cStr, uStr string
		if err := rows.Scan(&m.ID, &m.ProjectName, &m.Category, &m.Title, &m.TopicKey, &m.SummarySignature, &m.Tags, &cStr, &uStr); err != nil {
			return nil, err
		}
		m.CreatedAt = parseTime(cStr)
		m.UpdatedAt = parseTime(uStr)
		m.Source = s.source
		memories = append(memories, m)
	}

	return memories, nil
}

// PromoteMemory copies a memory from src Storage to dst Storage
func PromoteMemory(src *Storage, dst *Storage, id int64) (*core.Memory, error) {
	if src == nil || dst == nil {
		return nil, fmt.Errorf("almacenamiento no inicializado")
	}

	mem, err := src.GetMemoryByID(id)
	if err != nil {
		return nil, err
	}
	if mem == nil {
		return nil, fmt.Errorf("memoria #%d no encontrada en origen", id)
	}

	clone := &core.Memory{
		ProjectName:      mem.ProjectName,
		Category:         mem.Category,
		Title:            mem.Title,
		TopicKey:         mem.TopicKey,
		SummarySignature: mem.SummarySignature,
		Tags:             mem.Tags,
	}

	return dst.SaveMemory(clone)
}

// GetProjects returns all unique project names
func (s *Storage) GetProjects() ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT project_name FROM agent_memories ORDER BY project_name ASC")
	if err != nil {
		return nil, fmt.Errorf("error querying projects: %w", err)
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			projects = append(projects, p)
		}
	}
	return projects, nil
}

// GetStats returns usage metrics
func (s *Storage) GetStats() (*core.Stats, error) {
	query := `
		SELECT 
			COUNT(*) as total_memories,
			COUNT(DISTINCT project_name) as total_projects,
			COALESCE(SUM(LENGTH(summary_signature)), 0) as total_chars
		FROM agent_memories
	`
	var totalMemories, totalProjects, totalChars int64
	err := s.db.QueryRow(query).Scan(&totalMemories, &totalProjects, &totalChars)
	if err != nil {
		return nil, fmt.Errorf("error querying stats: %w", err)
	}

	return &core.Stats{
		TotalMemories:        totalMemories,
		TotalProjects:        totalProjects,
		EstimatedTokensSaved: totalChars / 4,
	}, nil
}

func parseTime(tStr string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, tStr); err == nil {
			return t
		}
	}
	return time.Now()
}
