package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/storage"
)

// Session represents an ephemeral one-time sharing session.
type Session struct {
	Code        string
	ProjectName string
	Port        int
	Addresses   []string
	doneCh      chan struct{}
	server      *http.Server
	listener    net.Listener
	mu          sync.Mutex
	closed      bool
}

// Close shuts down the ephemeral server immediately.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.doneCh)
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	return nil
}

// Done returns a channel that is closed when the session terminates or completes transfer.
func (s *Session) Done() <-chan struct{} {
	return s.doneCh
}


// StartShareSession starts an ephemeral one-time peer-to-peer session.
// Invariant: The session dies immediately upon transfer completion or timeout.
func StartShareSession(projectName string, memories []core.Memory, timeout time.Duration) (*Session, error) {
	if len(memories) == 0 {
		return nil, errors.New("no hay memorias para compartir en este proyecto")
	}

	code := GeneratePairCode()
	hostname, _ := os.Hostname()

	packet := &SyncPacket{
		Version:     "2.4.0",
		ProjectName: projectName,
		Sender:      hostname,
		Timestamp:   time.Now().UTC(),
		Memories:    memories,
		Code:        code,
	}

	rawJSON, err := json.Marshal(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal packet: %w", err)
	}

	// Encrypt packet with pairing code
	encryptedBytes, err := Encrypt(rawJSON, code)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt packet: %w", err)
	}

	// Bind ephemeral listener on all network interfaces
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind ephemeral port: %w", err)
	}

	tcpAddr := listener.Addr().(*net.TCPAddr)
	port := tcpAddr.Port

	session := &Session{
		Code:        code,
		ProjectName: projectName,
		Port:        port,
		Addresses:   getLocalIPs(port),
		doneCh:      make(chan struct{}),
		listener:    listener,
	}

	mux := http.NewServeMux()

	// One-time download endpoint
	mux.HandleFunc("/cogni/sync", func(w http.ResponseWriter, r *http.Request) {
		reqCode := r.Header.Get("X-Cogni-Code")
		if reqCode != code {
			http.Error(w, "Acceso no autorizado", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encryptedBytes)

		// Self-destruct session in background after delivery
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = session.Close()
		}()
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	session.server = server

	go func() {
		_ = server.Serve(listener)
	}()

	// Auto-expire session if no one connects within timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	go func() {
		select {
		case <-session.doneCh:
		case <-time.After(timeout):
			_ = session.Close()
		}
	}()

	return session, nil
}

// SyncFromPeer connects to a peer, downloads the encrypted memory bundle,
// decrypts with the pairing code, and merges memories non-destructively.
func SyncFromPeer(targetStorage *storage.Storage, code string, directAddr string) (*SyncResponse, error) {
	if !ValidatePairCode(code) {
		return nil, errors.New("formato de código de sincronización inválido (debe ser XXX-cogni-XX)")
	}

	var rawEncrypted []byte
	var fetchErr error

	// If direct address is provided (e.g., "192.168.1.15:52341" or "peer.tailscale.net:52341")
	if directAddr != "" {
		rawEncrypted, fetchErr = fetchDirect(directAddr, code)
	} else {
		// Scan localhost and common LAN fallback ports
		rawEncrypted, fetchErr = fetchLAN(code)
	}

	if fetchErr != nil {
		return nil, fmt.Errorf("no se pudo conectar con el compañero: %w", fetchErr)
	}

	// Decrypt payload with pairing code
	decryptedJSON, err := Decrypt(rawEncrypted, code)
	if err != nil {
		return nil, fmt.Errorf("error de descifrado: %w", err)
	}

	var packet SyncPacket
	if err := json.Unmarshal(decryptedJSON, &packet); err != nil {
		return nil, fmt.Errorf("paquete corrupto recibido: %w", err)
	}

	// Merge incoming memories non-destructively
	senderName := packet.Sender
	if senderName == "" {
		senderName = "peer"
	}

	return MergeMemories(targetStorage, packet.Memories, senderName)
}

func fetchDirect(addr, code string) ([]byte, error) {
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	url := strings.TrimRight(addr, "/") + "/cogni/sync"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Cogni-Code", code)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("código HTTP recibido: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func fetchLAN(code string) ([]byte, error) {
	return nil, errors.New("especifique la dirección del compañero con --from <host:puerto> o utilice el relay de internet")
}

func getLocalIPs(port int) []string {
	results := make([]string, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{fmt.Sprintf("127.0.0.1:%d", port)}
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip != nil {
				results = append(results, fmt.Sprintf("%s:%d", ip.String(), port))
			}
		}
	}

	if len(results) == 0 {
		results = append(results, fmt.Sprintf("127.0.0.1:%d", port))
	}
	return results
}
