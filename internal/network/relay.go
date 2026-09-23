package network

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// MaxPayloadSize: 512 KB is plenty for agent memories (~20 KB typical) while preventing memory exhaustion attacks
	MaxPayloadSize = 512 * 1024
	// DefaultTTL: 10 minutes lifespan before automatic purge
	DefaultTTL = 10 * time.Minute
	// MaxMemoryQueueSize: 50 MB total in RAM
	MaxMemoryQueueBytes = 50 * 1024 * 1024
)

// RelayDrop represents an ephemeral encrypted payload in memory.
type RelayDrop struct {
	CodeHash  string    `json:"code_hash"`
	Data      []byte    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	SenderIP  string    `json:"sender_ip"`
}

// RelayEvent records an audit event in the ring buffer.
type RelayEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "PUBLISH", "BURN", "EXPIRE", "PURGE"
	CodeHash  string    `json:"code_hash"`
	SizeBytes int       `json:"size_bytes"`
	SenderIP  string    `json:"sender_ip"`
	Detail    string    `json:"detail"`
}

// RelayMetrics snapshot for dashboard and API consumers.
type RelayMetrics struct {
	Service         string       `json:"service"`
	Version         string       `json:"version"`
	UptimeSeconds   int64        `json:"uptime_seconds"`
	ActiveDrops     int          `json:"active_drops"`
	BytesAllocated  int64        `json:"bytes_allocated"`
	MaxMemoryBytes  int64        `json:"max_memory_bytes"`
	TotalPublished  int64        `json:"total_published"`
	TotalConsumed   int64        `json:"total_consumed"`
	TotalExpired    int64        `json:"total_expired"`
	TotalBytesIn    int64        `json:"total_bytes_in"`
	TotalBytesOut   int64        `json:"total_bytes_out"`
	RecentEvents    []RelayEvent `json:"recent_events"`
	ActiveDropItems []RelayDrop  `json:"active_drop_items,omitempty"`
}

// RelayServer manages in-memory ephemeral dead drops with burn-after-reading.
type RelayServer struct {
	drops          sync.Map
	rateLimits     sync.Map
	totalBytes     int64
	totalPublished int64
	totalConsumed  int64
	totalExpired   int64
	totalBytesIn   int64
	totalBytesOut  int64
	startTime      time.Time
	events         []RelayEvent
	adminToken     string
	tokenFile      string
	mu             sync.Mutex
	stopCh         chan struct{}
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// NewRelayServer initializes a relay server and starts the periodic cleanup worker.
func NewRelayServer() *RelayServer {
	return NewRelayServerWithToken("")
}

// NewRelayServerWithToken initializes a relay server with an optional admin security token.
func NewRelayServerWithToken(adminToken string) *RelayServer {
	return NewRelayServerWithConfig(adminToken, "")
}

// NewRelayServerWithConfig initializes a relay server with token and dynamic token file for Infisical/secret managers.
func NewRelayServerWithConfig(adminToken, tokenFile string) *RelayServer {
	rs := &RelayServer{
		startTime:  time.Now().UTC(),
		events:     make([]RelayEvent, 0, 50),
		adminToken: strings.TrimSpace(adminToken),
		tokenFile:  strings.TrimSpace(tokenFile),
		stopCh:     make(chan struct{}),
	}
	go rs.cleanupWorker()
	return rs
}

// SetAdminToken sets or updates the administrative secret token.
func (rs *RelayServer) SetAdminToken(token string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.adminToken = strings.TrimSpace(token)
}

// SetTokenFile sets a dynamic file path where Infisical or other tools write the current secret token.
func (rs *RelayServer) SetTokenFile(path string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.tokenFile = strings.TrimSpace(path)
}

// getAdminToken retrieves the current active token dynamically (file -> env -> in-memory).
func (rs *RelayServer) getAdminToken() string {
	rs.mu.Lock()
	token := rs.adminToken
	tokenFile := rs.tokenFile
	rs.mu.Unlock()

	// 1. Check if token file exists and is readable (Hot reload from Infisical Agent)
	if tokenFile != "" {
		if data, err := os.ReadFile(tokenFile); err == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				return trimmed
			}
		}
	}

	// 2. Check dynamic environment variable
	if envToken := strings.TrimSpace(os.Getenv("COGNI_RELAY_ADMIN_TOKEN")); envToken != "" {
		return envToken
	}

	return token
}

func (rs *RelayServer) recordEvent(eventType, codeHash string, sizeBytes int, ip, detail string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	hashDisplay := codeHash
	if len(hashDisplay) > 12 {
		hashDisplay = hashDisplay[:12] + "..."
	}

	event := RelayEvent{
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		CodeHash:  hashDisplay,
		SizeBytes: sizeBytes,
		SenderIP:  ip,
		Detail:    detail,
	}

	// Keep last 50 events in a rolling slice
	if len(rs.events) >= 50 {
		rs.events = append(rs.events[1:], event)
	} else {
		rs.events = append(rs.events, event)
	}
}

func HashCode(code string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(code))))
	return hex.EncodeToString(h[:])
}

// Close stops the relay server background workers.
func (rs *RelayServer) Close() {
	close(rs.stopCh)
}

// cleanupWorker removes expired drops every 30 seconds.
func (rs *RelayServer) cleanupWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rs.stopCh:
			return
		case <-ticker.C:
			now := time.Now().UTC()
			rs.drops.Range(func(key, value any) bool {
				drop := value.(*RelayDrop)
				if now.After(drop.ExpiresAt) {
					rs.drops.Delete(key)
					rs.mu.Lock()
					rs.totalBytes -= int64(len(drop.Data))
					if rs.totalBytes < 0 {
						rs.totalBytes = 0
					}
					rs.totalExpired++
					rs.mu.Unlock()

					rs.recordEvent("EXPIRE", drop.CodeHash, len(drop.Data), drop.SenderIP, "Auto-purged after 10m TTL")
				}
				return true
			})

			// Clean expired rate limits
			rs.rateLimits.Range(func(key, value any) bool {
				entry := value.(*rateLimitEntry)
				if now.After(entry.resetTime) {
					rs.rateLimits.Delete(key)
				}
				return true
			})
		}
	}
}

// checkRateLimit enforces maximum 30 requests per minute per IP.
func (rs *RelayServer) checkRateLimit(ip string) bool {
	now := time.Now().UTC()
	val, loaded := rs.rateLimits.LoadOrStore(ip, &rateLimitEntry{
		count:     1,
		resetTime: now.Add(1 * time.Minute),
	})

	if !loaded {
		return true
	}

	entry := val.(*rateLimitEntry)
	if now.After(entry.resetTime) {
		entry.count = 1
		entry.resetTime = now.Add(1 * time.Minute)
		return true
	}

	entry.count++
	return entry.count <= 30
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (rs *RelayServer) verifyAdmin(r *http.Request) bool {
	token := rs.getAdminToken()
	if token == "" {
		return true // No token configured: open admin access
	}

	// 1. Check Header Authorization: Bearer <token>
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == token {
			return true
		}
	}
	// 2. Check Header X-Admin-Token
	if xToken := r.Header.Get("X-Admin-Token"); xToken == token {
		return true
	}
	// 3. Check Cookie cogni_token
	if cookie, err := r.Cookie("cogni_admin_token"); err == nil && cookie.Value == token {
		return true
	}
	// 4. Check Query param ?token=
	if qToken := r.URL.Query().Get("token"); qToken == token {
		return true
	}

	return false
}

// Handler returns the HTTP handler with all endpoints and security middlewares.
func (rs *RelayServer) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", rs.handleDashboard)
	mux.HandleFunc("/admin", rs.handleDashboard)
	mux.HandleFunc("/health", rs.handleHealth)
	mux.HandleFunc("/api/v1/metrics", rs.handleMetrics)
	mux.HandleFunc("/api/v1/admin/purge", rs.handlePurge)
	mux.HandleFunc("/api/v1/drop", rs.handleDrop)
	mux.HandleFunc("/api/v1/drop/", rs.handleFetch)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

		// Rate limiting only for public API drop endpoints, not dashboard
		if strings.HasPrefix(r.URL.Path, "/api/v1/drop") {
			ip := extractIP(r)
			if !rs.checkRateLimit(ip) {
				http.Error(w, `{"error":"Rate limit excedido. Espere 1 minuto."}`, http.StatusTooManyRequests)
				return
			}
		}

		mux.ServeHTTP(w, r)
	})
}

func (rs *RelayServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	bytesAllocated := rs.totalBytes
	rs.mu.Unlock()

	activeDrops := 0
	rs.drops.Range(func(_, _ any) bool {
		activeDrops++
		return true
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":          "ok",
		"service":         "cogni-relay",
		"active_drops":    activeDrops,
		"bytes_allocated": bytesAllocated,
		"max_bytes":       MaxMemoryQueueBytes,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	})
}

func (rs *RelayServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if !rs.verifyAdmin(r) {
		http.Error(w, `{"error":"Unauthorized. Token admin requerido."}`, http.StatusUnauthorized)
		return
	}

	rs.mu.Lock()
	bytesAllocated := rs.totalBytes
	totalPub := rs.totalPublished
	totalCons := rs.totalConsumed
	totalExp := rs.totalExpired
	bytesIn := rs.totalBytesIn
	bytesOut := rs.totalBytesOut
	startTime := rs.startTime
	eventsCopy := make([]RelayEvent, len(rs.events))
	copy(eventsCopy, rs.events)
	rs.mu.Unlock()

	activeDrops := 0
	activeList := make([]RelayDrop, 0)
	rs.drops.Range(func(_, val any) bool {
		activeDrops++
		d := val.(*RelayDrop)
		activeList = append(activeList, RelayDrop{
			CodeHash:  d.CodeHash[:12] + "...",
			CreatedAt: d.CreatedAt,
			ExpiresAt: d.ExpiresAt,
			SenderIP:  d.SenderIP,
		})
		return true
	})

	uptime := int64(time.Since(startTime).Seconds())

	metrics := RelayMetrics{
		Service:         "cogni-relay",
		Version:         "v2.4.0",
		UptimeSeconds:   uptime,
		ActiveDrops:     activeDrops,
		BytesAllocated:  bytesAllocated,
		MaxMemoryBytes:  MaxMemoryQueueBytes,
		TotalPublished:  totalPub,
		TotalConsumed:   totalCons,
		TotalExpired:    totalExp,
		TotalBytesIn:    bytesIn,
		TotalBytesOut:   bytesOut,
		RecentEvents:    eventsCopy,
		ActiveDropItems: activeList,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

func (rs *RelayServer) handlePurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	if !rs.verifyAdmin(r) {
		http.Error(w, `{"error":"Unauthorized. Token admin requerido."}`, http.StatusUnauthorized)
		return
	}

	purgedCount := 0
	var freedBytes int64

	rs.drops.Range(func(key, val any) bool {
		drop := val.(*RelayDrop)
		rs.drops.Delete(key)
		purgedCount++
		freedBytes += int64(len(drop.Data))
		return true
	})

	rs.mu.Lock()
	rs.totalBytes = 0
	rs.mu.Unlock()

	rs.recordEvent("PURGE", "ALL", int(freedBytes), extractIP(r), "Manual purge via admin dashboard")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":       true,
		"purged_count":  purgedCount,
		"freed_bytes":   freedBytes,
		"message":       "Todo el almacenamiento efímero en RAM ha sido vaciado exitosamente.",
	})
}

func (rs *RelayServer) handleDrop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	code := r.Header.Get("X-Cogni-Code")
	if !ValidatePairCode(code) {
		http.Error(w, "Código inválido. Formato requerido: XXX-cogni-XX", http.StatusBadRequest)
		return
	}

	rs.mu.Lock()
	if rs.totalBytes > MaxMemoryQueueBytes {
		rs.mu.Unlock()
		http.Error(w, "Capacidad máxima del servidor alcanzada temporalmente. Intente más tarde.", http.StatusServiceUnavailable)
		return
	}
	rs.mu.Unlock()

	// Enforce 512 KB payload limit
	limitedReader := io.LimitReader(r.Body, MaxPayloadSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		http.Error(w, "Error leyendo cuerpo de petición", http.StatusInternalServerError)
		return
	}
	if len(data) > MaxPayloadSize {
		http.Error(w, "Carga demasiado grande. Límite máximo: 512 KB", http.StatusRequestEntityTooLarge)
		return
	}
	if len(data) == 0 {
		http.Error(w, "Carga vacía", http.StatusBadRequest)
		return
	}

	codeHash := HashCode(code)
	now := time.Now().UTC()
	senderIP := extractIP(r)
	drop := &RelayDrop{
		CodeHash:  codeHash,
		Data:      data,
		CreatedAt: now,
		ExpiresAt: now.Add(DefaultTTL),
		SenderIP:  senderIP,
	}

	rs.drops.Store(codeHash, drop)

	rs.mu.Lock()
	rs.totalBytes += int64(len(data))
	rs.totalBytesIn += int64(len(data))
	rs.totalPublished++
	rs.mu.Unlock()

	rs.recordEvent("PUBLISH", codeHash, len(data), senderIP, "Uploaded encrypted payload")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":    true,
		"code_hash":  codeHash[:12] + "...",
		"expires_in": "10m",
		"size_bytes": len(data),
		"message":    "Paquete cifrado recibido. Se autodestruirá al primer consumo o tras 10 minutos.",
	})
}

func (rs *RelayServer) handleFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/api/v1/drop/")
	code = strings.TrimSpace(code)
	if !ValidatePairCode(code) {
		http.Error(w, "Formato de código inválido", http.StatusBadRequest)
		return
	}

	codeHash := HashCode(code)
	senderIP := extractIP(r)

	// Atomic Burn-After-Reading: load and delete in one step
	val, loaded := rs.drops.LoadAndDelete(codeHash)
	if !loaded {
		http.Error(w, "Paquete no encontrado, expirado o ya consumido", http.StatusNotFound)
		return
	}

	drop := val.(*RelayDrop)

	rs.mu.Lock()
	rs.totalBytes -= int64(len(drop.Data))
	if rs.totalBytes < 0 {
		rs.totalBytes = 0
	}
	rs.totalBytesOut += int64(len(drop.Data))
	rs.totalConsumed++
	rs.mu.Unlock()

	rs.recordEvent("BURN", codeHash, len(drop.Data), senderIP, "Downloaded and burned from RAM")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="bundle.cogni"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(drop.Data)
}

func (rs *RelayServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Cogni Relay - Dashboard & Telemetria</title>
  <style>
    :root {
      --bg: #090d16;
      --card-bg: #111827;
      --card-border: #1f2937;
      --text-main: #f3f4f6;
      --text-muted: #9ca3af;
      --accent: #3b82f6;
      --accent-hover: #2563eb;
      --success: #10b981;
      --warning: #f59e0b;
      --danger: #ef4444;
      --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text-main);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      padding: 24px;
      line-height: 1.5;
    }
    .container { max-width: 1200px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-bottom: 20px;
      border-bottom: 1px solid var(--card-border);
      margin-bottom: 24px;
      flex-wrap: wrap;
      gap: 12px;
    }
    .logo-group { display: flex; align-items: center; gap: 12px; }
    .logo-title { font-size: 20px; font-weight: 700; letter-spacing: -0.5px; }
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      background: rgba(16, 185, 129, 0.1);
      border: 1px solid var(--success);
      color: var(--success);
      border-radius: 9999px;
      font-size: 12px;
      font-weight: 600;
    }
    .status-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--success); animation: pulse 2s infinite; }
    @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
    .btn-group { display: flex; gap: 10px; align-items: center; }
    button {
      padding: 8px 14px;
      border-radius: 6px;
      border: none;
      cursor: pointer;
      font-size: 13px;
      font-weight: 600;
      transition: all 0.2s;
    }
    .btn-primary { background: var(--accent); color: #fff; }
    .btn-primary:hover { background: var(--accent-hover); }
    .btn-danger { background: rgba(239, 68, 68, 0.15); border: 1px solid var(--danger); color: var(--danger); }
    .btn-danger:hover { background: var(--danger); color: #fff; }
    .btn-secondary { background: #374151; color: #f3f4f6; }
    .btn-secondary:hover { background: #4b5563; }
    .grid-cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 8px;
      padding: 16px 20px;
    }
    .card-title { font-size: 12px; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
    .card-value { font-size: 28px; font-weight: 700; color: var(--text-main); font-family: var(--font-mono); }
    .card-sub { font-size: 12px; color: var(--text-muted); margin-top: 4px; }
    .section-title { font-size: 16px; font-weight: 600; margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center; }
    .table-container {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 8px;
      overflow-x: auto;
      margin-bottom: 24px;
    }
    table { width: 100%; border-collapse: collapse; font-size: 13px; text-align: left; }
    th { padding: 12px 16px; background: rgba(255,255,255,0.02); border-bottom: 1px solid var(--card-border); color: var(--text-muted); font-weight: 600; }
    td { padding: 12px 16px; border-bottom: 1px solid var(--card-border); font-family: var(--font-mono); font-size: 12px; }
    tr:last-child td { border-bottom: none; }
    .tag { display: inline-block; padding: 2px 6px; border-radius: 4px; font-size: 11px; font-weight: 700; }
    .tag-pub { background: rgba(59, 130, 246, 0.2); color: var(--accent); }
    .tag-burn { background: rgba(16, 185, 129, 0.2); color: var(--success); }
    .tag-exp { background: rgba(245, 158, 11, 0.2); color: var(--warning); }
    .tag-purge { background: rgba(239, 68, 68, 0.2); color: var(--danger); }
    .empty-state { padding: 24px; text-align: center; color: var(--text-muted); font-size: 13px; }
    .toast {
      position: fixed; bottom: 20px; right: 20px; background: var(--card-bg); border: 1px solid var(--card-border);
      padding: 12px 20px; border-radius: 6px; font-size: 13px; box-shadow: 0 4px 12px rgba(0,0,0,0.5);
      transform: translateY(100px); opacity: 0; transition: all 0.3s ease; z-index: 1000;
    }
    .toast.show { transform: translateY(0); opacity: 1; }
    .login-box {
      max-width: 420px; margin: 80px auto; background: var(--card-bg); border: 1px solid var(--card-border);
      border-radius: 12px; padding: 32px; box-shadow: 0 8px 30px rgba(0,0,0,0.4); text-align: center;
    }
    .login-title { font-size: 18px; font-weight: 700; margin-bottom: 8px; }
    .login-desc { font-size: 13px; color: var(--text-muted); margin-bottom: 24px; }
    .login-input {
      width: 100%; padding: 12px 14px; background: #0b111e; border: 1px solid #374151;
      border-radius: 6px; color: #fff; font-size: 14px; font-family: var(--font-mono); margin-bottom: 16px;
    }
    .login-input:focus { outline: none; border-color: var(--accent); }
  </style>
</head>
<body>
  <div id="login-container" class="login-box" style="display: none;">
    <div class="login-title">Panel de Administracion Protegido</div>
    <div class="login-desc">Ingrese el Token de Seguridad de Cogni Relay para ver metricas y gestionar el almacenamiento.</div>
    <form onsubmit="handleLogin(event)">
      <input type="password" id="input-token" class="login-input" placeholder="COGNI_RELAY_ADMIN_TOKEN" autofocus required>
      <button type="submit" class="btn-primary" style="width: 100%; padding: 12px;">Acceder al Dashboard</button>
    </form>
  </div>

  <div id="dashboard-container" class="container" style="display: none;">
    <header>
      <div class="logo-group">
        <span class="logo-title">Cogni Relay Admin & Telemetry</span>
        <span class="status-badge"><span class="status-dot"></span> OPERATIONAL (RAM)</span>
      </div>
      <div class="btn-group">
        <span id="uptime-label" style="font-size: 12px; color: var(--text-muted); margin-right: 8px;">Uptime: --</span>
        <button class="btn-danger" onclick="purgeStorage()">Vaciar Storage (Purge)</button>
        <button class="btn-primary" onclick="fetchMetrics()">Actualizar</button>
        <button id="btn-logout" class="btn-secondary" style="display: none;" onclick="handleLogout()">Salir</button>
      </div>
    </header>

    <div class="grid-cards">
      <div class="card">
        <div class="card-title">Drops Publicados vs Consumidos</div>
        <div class="card-value" id="val-pub-cons">0 / 0</div>
        <div class="card-sub" id="sub-expired">Expirados por TTL: 0</div>
      </div>
      <div class="card">
        <div class="card-title">Almacenamiento RAM en Uso</div>
        <div class="card-value" id="val-ram-bytes">0 KB</div>
        <div class="card-sub">Tope Maximo: 50 MB</div>
      </div>
      <div class="card">
        <div class="card-title">Drops Activos en Espera</div>
        <div class="card-value" id="val-active-drops" style="color: var(--accent);">0</div>
        <div class="card-sub">En cola de memoria efimera</div>
      </div>
      <div class="card">
        <div class="card-title">Trafico Total Transferido</div>
        <div class="card-value" id="val-traffic">0 KB</div>
        <div class="card-sub" id="sub-traffic-split">In: 0 KB | Out: 0 KB</div>
      </div>
    </div>

    <div class="section-title">
      <span>Drops Activos en Memoria (Esperando Lectura)</span>
    </div>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Hash de Codigo</th>
            <th>IP Origen</th>
            <th>Creado</th>
            <th>Expira</th>
            <th>Estado</th>
          </tr>
        </thead>
        <tbody id="active-table-body">
          <tr><td colspan="5" class="empty-state">No hay paquetes en memoria actualmente.</td></tr>
        </tbody>
      </table>
    </div>

    <div class="section-title">
      <span>Registro de Actividad Reciente (Ultimas Transacciones)</span>
    </div>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Hora (UTC)</th>
            <th>Evento</th>
            <th>Hash Codigo</th>
            <th>Tamano</th>
            <th>IP Cliente</th>
            <th>Detalle</th>
          </tr>
        </thead>
        <tbody id="events-table-body">
          <tr><td colspan="6" class="empty-state">No hay eventos registrados aun.</td></tr>
        </tbody>
      </table>
    </div>
  </div>

  <div id="toast" class="toast"></div>

  <script>
    function getToken() {
      const urlParams = new URLSearchParams(window.location.search);
      return urlParams.get('token') || sessionStorage.getItem('cogni_admin_token') || '';
    }

    function setToken(token) {
      if (token) {
        sessionStorage.setItem('cogni_admin_token', token);
      } else {
        sessionStorage.removeItem('cogni_admin_token');
      }
    }

    function formatBytes(bytes) {
      if (!bytes || bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    function formatUptime(seconds) {
      const d = Math.floor(seconds / (3600*24));
      const h = Math.floor(seconds % (3600*24) / 3600);
      const m = Math.floor(seconds % 3600 / 60);
      const s = Math.floor(seconds % 60);
      if (d > 0) return d + 'd ' + h + 'h ' + m + 'm';
      if (h > 0) return h + 'h ' + m + 'm ' + s + 's';
      return m + 'm ' + s + 's';
    }

    function showToast(msg) {
      const toast = document.getElementById('toast');
      toast.innerText = msg;
      toast.classList.add('show');
      setTimeout(() => toast.classList.remove('show'), 3000);
    }

    async function handleLogin(e) {
      e.preventDefault();
      const token = document.getElementById('input-token').value.trim();
      if (!token) return;

      setToken(token);
      await fetchMetrics();
    }

    function handleLogout() {
      setToken('');
      document.getElementById('dashboard-container').style.display = 'none';
      document.getElementById('login-container').style.display = 'block';
      document.getElementById('input-token').value = '';
      showToast('Sesion cerrada.');
    }

    async function fetchMetrics() {
      try {
        const token = getToken();
        const headers = {};
        if (token) {
          headers['Authorization'] = 'Bearer ' + token;
        }

        const res = await fetch('/api/v1/metrics', { headers: headers });
        if (!res.ok) {
          if (res.status === 401) {
            document.getElementById('dashboard-container').style.display = 'none';
            document.getElementById('login-container').style.display = 'block';
            if (token) {
              showToast('Token invalido o expirado.');
              setToken('');
            }
            return;
          }
          showToast('Error al obtener metricas: ' + res.statusText);
          return;
        }

        const data = await res.json();
        document.getElementById('login-container').style.display = 'none';
        document.getElementById('dashboard-container').style.display = 'block';
        if (token) {
          document.getElementById('btn-logout').style.display = 'inline-block';
        }
        render(data);
      } catch (err) {
        console.error('Error fetching metrics:', err);
      }
    }

    function render(data) {
      document.getElementById('uptime-label').innerText = 'Uptime: ' + formatUptime(data.uptime_seconds);
      document.getElementById('val-pub-cons').innerText = data.total_published + ' / ' + data.total_consumed;
      document.getElementById('sub-expired').innerText = 'Expirados por TTL: ' + data.total_expired;
      document.getElementById('val-ram-bytes').innerText = formatBytes(data.bytes_allocated);
      document.getElementById('val-active-drops').innerText = data.active_drops;
      document.getElementById('val-traffic').innerText = formatBytes(data.total_bytes_in + data.total_bytes_out);
      document.getElementById('sub-traffic-split').innerText = 'In: ' + formatBytes(data.total_bytes_in) + ' | Out: ' + formatBytes(data.total_bytes_out);

      // Render Active Drops
      const activeTbody = document.getElementById('active-table-body');
      if (!data.active_drop_items || data.active_drop_items.length === 0) {
        activeTbody.innerHTML = '<tr><td colspan="5" class="empty-state">No hay paquetes en memoria actualmente.</td></tr>';
      } else {
        activeTbody.innerHTML = data.active_drop_items.map(function(d) {
          return '<tr>' +
            '<td><code>' + (d.code_hash || '-') + '</code></td>' +
            '<td>' + (d.sender_ip || 'local') + '</td>' +
            '<td>' + new Date(d.created_at).toLocaleTimeString() + '</td>' +
            '<td>' + new Date(d.expires_at).toLocaleTimeString() + '</td>' +
            '<td><span class="tag tag-pub">PENDIENTE</span></td>' +
          '</tr>';
        }).join('');
      }

      // Render Events
      const eventsTbody = document.getElementById('events-table-body');
      if (!data.recent_events || data.recent_events.length === 0) {
        eventsTbody.innerHTML = '<tr><td colspan="6" class="empty-state">No hay eventos registrados aun.</td></tr>';
      } else {
        const sorted = data.recent_events.slice().reverse();
        eventsTbody.innerHTML = sorted.map(function(ev) {
          var tagClass = 'tag-pub';
          if (ev.type === 'BURN') tagClass = 'tag-burn';
          if (ev.type === 'EXPIRE') tagClass = 'tag-exp';
          if (ev.type === 'PURGE') tagClass = 'tag-purge';

          return '<tr>' +
            '<td>' + new Date(ev.timestamp).toLocaleTimeString() + '</td>' +
            '<td><span class="tag ' + tagClass + '">' + ev.type + '</span></td>' +
            '<td><code>' + (ev.code_hash || '-') + '</code></td>' +
            '<td>' + formatBytes(ev.size_bytes) + '</td>' +
            '<td>' + (ev.sender_ip || 'local') + '</td>' +
            '<td>' + ev.detail + '</td>' +
          '</tr>';
        }).join('');
      }
    }

    async function purgeStorage() {
      if (!confirm('Esta seguro de que desea vaciar todo el almacenamiento efimero de la memoria RAM?')) {
        return;
      }
      try {
        const token = getToken();
        const headers = {};
        if (token) {
          headers['Authorization'] = 'Bearer ' + token;
        }

        const res = await fetch('/api/v1/admin/purge', { method: 'POST', headers: headers });
        if (res.ok) {
          const json = await res.json();
          showToast(json.message || 'Storage vaciado exitosamente');
          fetchMetrics();
        } else {
          showToast('Error al vaciar storage: ' + res.statusText);
        }
      } catch (err) {
        showToast('Fallo en la peticion');
      }
    }

    fetchMetrics();
    setInterval(fetchMetrics, 3000);
  </script>
</body>
</html>`
