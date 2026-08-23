package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/storage"
	"github.com/AdelysAlberto/cogni/web"
)

type Server struct {
	localStorage  *storage.Storage
	globalStorage *storage.Storage
	host          string
	port          int
	srv           *http.Server
}

func New(localStorage, globalStorage *storage.Storage, host string, port int) *Server {
	if host == "" {
		host = "127.0.0.1"
	}
	if port <= 0 {
		port = 3000
	}
	return &Server{
		localStorage:  localStorage,
		globalStorage: globalStorage,
		host:          host,
		port:          port,
	}
}

// Start launches the HTTP server with automatic port recovery if target port is in use
func (s *Server) Start(openBrowser bool) (string, error) {
	mux := http.NewServeMux()

	// 1. Register API Handlers
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/memories", s.handleMemories)
	mux.HandleFunc("/api/memories/promote", s.handlePromote)
	mux.HandleFunc("/api/memories/", s.handleMemoryByID)

	// 2. Register Embedded Static Web Files
	fs, err := web.GetFileSystem()
	if err != nil {
		return "", fmt.Errorf("failed to load embedded UI filesystem: %w", err)
	}
	fileServer := http.FileServer(fs)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Avoid directory listing, fallback to index
		if r.URL.Path == "/" || !strings.Contains(r.URL.Path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	// Wrap with security middleware
	handler := s.securityMiddleware(mux)

	// Dynamic Port Finding to prevent EADDRINUSE
	listener, finalPort, err := findAvailableListener(s.host, s.port)
	if err != nil {
		return "", fmt.Errorf("could not find available port: %w", err)
	}
	s.port = finalPort

	s.srv = &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	url := fmt.Sprintf("http://%s:%d", s.host, s.port)
	fmt.Printf("\n🧠 [Cogni] Dashboard iniciado con éxito en: %s\n", url)
	if s.localStorage != nil {
		fmt.Printf("📂 Base de datos local:  %s\n", s.localStorage.DBPath())
	}
	if s.globalStorage != nil {
		fmt.Printf("🌐 Base de datos global: %s\n", s.globalStorage.DBPath())
	}
	fmt.Println("Presiona Ctrl+C para detener el servidor.")
	fmt.Println()

	if openBrowser {
		go openBrowserURL(url)
	}

	return url, s.srv.Serve(listener)
}

func findAvailableListener(host string, startPort int) (net.Listener, int, error) {
	for p := startPort; p < startPort+50; p++ {
		addr := fmt.Sprintf("%s:%d", host, p)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			return l, p, nil
		}
	}
	l, err := net.Listen("tcp", fmt.Sprintf("%s:0", host))
	if err != nil {
		return nil, 0, err
	}
	addr := l.Addr().(*net.TCPAddr)
	return l, addr.Port, nil
}

func openBrowserURL(url string) {
	time.Sleep(200 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostHeader := r.Host
		if strings.Contains(hostHeader, ":") {
			hostHeader = strings.Split(hostHeader, ":")[0]
		}

		if s.host == "127.0.0.1" && hostHeader != "localhost" && hostHeader != "127.0.0.1" && hostHeader != "" {
			http.Error(w, "Acceso denegado: Host no autorizado.", http.StatusForbidden)
			return
		}

		origin := r.Header.Get("Origin")
		if origin != "" {
			if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			} else {
				http.Error(w, `{"error":"CORS no autorizado"}`, http.StatusForbidden)
				return
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var totalMemories, totalProjects, totalTokens int64
	seenProjects := make(map[string]bool)

	if s.localStorage != nil {
		if st, err := s.localStorage.GetStats(); err == nil {
			totalMemories += st.TotalMemories
			totalTokens += st.EstimatedTokensSaved
		}
		if projs, err := s.localStorage.GetProjects(); err == nil {
			for _, p := range projs {
				seenProjects[p] = true
			}
		}
	}

	if s.globalStorage != nil && (s.localStorage == nil || s.localStorage.DBPath() != s.globalStorage.DBPath()) {
		if st, err := s.globalStorage.GetStats(); err == nil {
			totalMemories += st.TotalMemories
			totalTokens += st.EstimatedTokensSaved
		}
		if projs, err := s.globalStorage.GetProjects(); err == nil {
			for _, p := range projs {
				seenProjects[p] = true
			}
		}
	}

	totalProjects = int64(len(seenProjects))

	sendJSON(w, core.Stats{
		TotalMemories:        totalMemories,
		TotalProjects:        totalProjects,
		EstimatedTokensSaved: totalTokens,
	}, http.StatusOK)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectMap := make(map[string]bool)
	if s.localStorage != nil {
		if projs, err := s.localStorage.GetProjects(); err == nil {
			for _, p := range projs {
				projectMap[p] = true
			}
		}
	}
	if s.globalStorage != nil {
		if projs, err := s.globalStorage.GetProjects(); err == nil {
			for _, p := range projs {
				projectMap[p] = true
			}
		}
	}

	var projects []string
	for p := range projectMap {
		projects = append(projects, p)
	}
	sort.Strings(projects)

	sendJSON(w, projects, http.StatusOK)
}

func (s *Server) handleMemories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query().Get("query")
		project := r.URL.Query().Get("project")
		category := r.URL.Query().Get("category")
		sourceFilter := r.URL.Query().Get("source") // "local", "global", or ""
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 100
		}

		var combined []core.Memory

		// Query Local Storage
		if (sourceFilter == "" || sourceFilter == "local") && s.localStorage != nil {
			var localMems []core.Memory
			if query != "" {
				localMems, _ = s.localStorage.SearchMemories(project, query, category, limit)
			} else {
				localMems, _ = s.localStorage.ListMemories(project, category, limit, 0)
			}
			for i := range localMems {
				localMems[i].Source = "local"
			}
			combined = append(combined, localMems...)
		}

		// Query Global Storage
		if (sourceFilter == "" || sourceFilter == "global") && s.globalStorage != nil {
			// Avoid duplicate query if local and global point to same DB
			if s.localStorage == nil || s.localStorage.DBPath() != s.globalStorage.DBPath() {
				var globalMems []core.Memory
				if query != "" {
					globalMems, _ = s.globalStorage.SearchMemories(project, query, category, limit)
				} else {
					globalMems, _ = s.globalStorage.ListMemories(project, category, limit, 0)
				}
				for i := range globalMems {
					globalMems[i].Source = "global"
				}
				combined = append(combined, globalMems...)
			}
		}

		// Sort all memories by created_at DESC
		sort.Slice(combined, func(i, j int) bool {
			return combined[i].CreatedAt.After(combined[j].CreatedAt)
		})

		if len(combined) > limit {
			combined = combined[:limit]
		}

		if combined == nil {
			combined = []core.Memory{}
		}
		sendJSON(w, combined, http.StatusOK)

	case http.MethodPost:
		var m struct {
			ProjectName      string `json:"project_name"`
			Category         string `json:"category"`
			Title            string `json:"title"`
			SummarySignature string `json:"summary_signature"`
			Tags             string `json:"tags"`
			Target           string `json:"target"` // "local" or "global"
		}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			sendJSON(w, map[string]string{"error": "Payload JSON inválido"}, http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(m.Title) == "" || strings.TrimSpace(m.SummarySignature) == "" {
			sendJSON(w, map[string]string{"error": "Campos 'title' y 'summary_signature' son obligatorios"}, http.StatusBadRequest)
			return
		}

		if m.ProjectName == "" {
			m.ProjectName = "general"
		}
		if m.Category == "" {
			m.Category = "general"
		}
		formattedTags := core.FormatTags(m.Tags, m.ProjectName)

		mem := &core.Memory{
			ProjectName:      m.ProjectName,
			Category:         m.Category,
			Title:            m.Title,
			SummarySignature: m.SummarySignature,
			Tags:             formattedTags,
		}

		targetStorage := s.localStorage
		if m.Target == "global" || targetStorage == nil {
			targetStorage = s.globalStorage
		}

		if targetStorage == nil {
			sendJSON(w, map[string]string{"error": "Base de datos de destino no disponible"}, http.StatusInternalServerError)
			return
		}

		saved, err := targetStorage.SaveMemory(mem)
		if err != nil {
			sendJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		saved.Source = targetStorage.Source()
		sendJSON(w, saved, http.StatusCreated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePromote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID   int64  `json:"id"`
		From string `json:"from"` // "local" or "global"
		To   string `json:"to"`   // "global" or "local"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, map[string]string{"error": "Payload inválido"}, http.StatusBadRequest)
		return
	}

	if req.ID <= 0 {
		sendJSON(w, map[string]string{"error": "ID es obligatorio"}, http.StatusBadRequest)
		return
	}

	var src, dst *storage.Storage
	if req.From == "global" || (req.To == "local" && s.localStorage != nil) {
		src = s.globalStorage
		dst = s.localStorage
	} else {
		src = s.localStorage
		dst = s.globalStorage
	}

	if src == nil || dst == nil {
		sendJSON(w, map[string]string{"error": "Ambas bases de datos (local y global) deben estar activas para promover"}, http.StatusBadRequest)
		return
	}

	promoted, err := storage.PromoteMemory(src, dst, req.ID)
	if err != nil {
		sendJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	promoted.Source = dst.Source()
	sendJSON(w, promoted, http.StatusOK)
}

func (s *Server) handleMemoryByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/memories/")
	if path == "promote" {
		s.handlePromote(w, r)
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil || id <= 0 {
		sendJSON(w, map[string]string{"error": "ID de memoria inválido"}, http.StatusBadRequest)
		return
	}

	source := r.URL.Query().Get("source") // "local" or "global"
	targetStorage := s.localStorage
	if source == "global" || targetStorage == nil {
		targetStorage = s.globalStorage
	}

	switch r.Method {
	case http.MethodGet:
		mem, err := targetStorage.GetMemoryByID(id)
		if err != nil {
			sendJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		if mem == nil && targetStorage != s.globalStorage && s.globalStorage != nil {
			mem, err = s.globalStorage.GetMemoryByID(id)
		}
		if mem == nil {
			sendJSON(w, map[string]string{"error": "Memoria no encontrada"}, http.StatusNotFound)
			return
		}
		sendJSON(w, mem, http.StatusOK)

	case http.MethodPut:
		var body struct {
			Title            string `json:"title"`
			TopicKey         string `json:"topic_key"`
			SummarySignature string `json:"summary_signature"`
			Category         string `json:"category"`
			Tags             string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			sendJSON(w, map[string]string{"error": "Payload JSON inválido"}, http.StatusBadRequest)
			return
		}

		updated, err := targetStorage.UpdateMemory(id, body.Title, body.SummarySignature, body.Category, body.Tags, body.TopicKey)
		if err != nil {
			sendJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		sendJSON(w, updated, http.StatusOK)

	case http.MethodDelete:
		deleted, err := targetStorage.DeleteMemory(id)
		if err != nil {
			sendJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		sendJSON(w, map[string]bool{"deleted": deleted}, http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func sendJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
