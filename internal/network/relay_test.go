package network

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRelayServerDropAndBurnAfterReading(t *testing.T) {
	rs := NewRelayServer()
	defer rs.Close()

	handler := rs.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	code := "381-c0gn1-42"
	payload := []byte("Paquete cifrado simulado con memorias de agente.")

	// 1. Test POST /api/v1/drop with valid code
	req, err := http.NewRequest("POST", server.URL+"/api/v1/drop", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Cogni-Code", code)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/drop failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Test GET /api/v1/drop/{code} (First consumption -> SUCCESS)
	fetchReq, err := http.NewRequest("GET", server.URL+"/api/v1/drop/"+code, nil)
	if err != nil {
		t.Fatalf("Failed to create fetch request: %v", err)
	}

	fetchResp, err := http.DefaultClient.Do(fetchReq)
	if err != nil {
		t.Fatalf("GET /api/v1/drop failed: %v", err)
	}
	if fetchResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK on first fetch, got %d", fetchResp.StatusCode)
	}

	downloaded, err := io.ReadAll(fetchResp.Body)
	fetchResp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(downloaded) != string(payload) {
		t.Errorf("Downloaded payload mismatch: expected '%s', got '%s'", string(payload), string(downloaded))
	}

	// 3. Invariant: Second GET MUST return 404 (Burn-After-Reading)
	secondResp, err := http.DefaultClient.Do(fetchReq)
	if err != nil {
		t.Fatalf("Second fetch failed: %v", err)
	}
	defer secondResp.Body.Close()

	if secondResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found on second fetch (Burn-After-Reading), got %d", secondResp.StatusCode)
	}
}

func TestRelayServerMaxPayloadRejection(t *testing.T) {
	rs := NewRelayServer()
	defer rs.Close()

	handler := rs.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	code := "555-k0gni-99"
	// Create payload larger than 512 KB
	oversized := make([]byte, 513*1024)

	req, _ := http.NewRequest("POST", server.URL+"/api/v1/drop", bytes.NewReader(oversized))
	req.Header.Set("X-Cogni-Code", code)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("Expected 413 Payload Too Large, got %d", resp.StatusCode)
	}
}

func TestRelayServerDashboardAndMetrics(t *testing.T) {
	rs := NewRelayServer()
	defer rs.Close()

	handler := rs.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test GET / (Dashboard HTML)
	dashResp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer dashResp.Body.Close()
	if dashResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for dashboard, got %d", dashResp.StatusCode)
	}

	body, _ := io.ReadAll(dashResp.Body)
	if !bytes.Contains(body, []byte("Cogni Relay")) {
		t.Fatalf("Dashboard HTML missing expected title")
	}

	// Test GET /api/v1/metrics
	metricsResp, err := http.Get(server.URL + "/api/v1/metrics")
	if err != nil {
		t.Fatalf("GET /api/v1/metrics failed: %v", err)
	}
	defer metricsResp.Body.Close()
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for metrics, got %d", metricsResp.StatusCode)
	}
}

func TestRelayServerAdminPurgeAndToken(t *testing.T) {
	secretToken := "super-secret-token"
	rs := NewRelayServerWithToken(secretToken)
	defer rs.Close()

	handler := rs.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	code := "123-c0gn1-45"
	payload := []byte("sample payload")

	// Post a drop
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/drop", bytes.NewReader(payload))
	req.Header.Set("X-Cogni-Code", code)
	postResp, err := http.DefaultClient.Do(req)
	if err != nil || postResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to post drop: %v", err)
	}
	postResp.Body.Close()

	// 1. Unauthenticated purge MUST fail with 401
	unauthPurgeReq, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/purge", nil)
	unauthResp, err := http.DefaultClient.Do(unauthPurgeReq)
	if err != nil {
		t.Fatalf("Unauth purge request failed: %v", err)
	}
	defer unauthResp.Body.Close()
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for unauth purge, got %d", unauthResp.StatusCode)
	}

	// 2. Authenticated purge with Header MUST succeed with 200
	authPurgeReq, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/purge", nil)
	authPurgeReq.Header.Set("Authorization", "Bearer "+secretToken)
	authResp, err := http.DefaultClient.Do(authPurgeReq)
	if err != nil {
		t.Fatalf("Auth purge request failed: %v", err)
	}
	defer authResp.Body.Close()
	if authResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for auth purge, got %d", authResp.StatusCode)
	}

	// 3. Drop must now be gone (404)
	fetchReq, _ := http.NewRequest("GET", server.URL+"/api/v1/drop/"+code, nil)
	fetchResp, err := http.DefaultClient.Do(fetchReq)
	if err != nil {
		t.Fatalf("Fetch after purge failed: %v", err)
	}
	defer fetchResp.Body.Close()
	if fetchResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found after purge, got %d", fetchResp.StatusCode)
	}
}

func TestRelayServerDynamicTokenFileHotReload(t *testing.T) {
	// Create temporary token file
	tmpFile, err := os.CreateTemp("", "cogni_token_*.secret")
	if err != nil {
		t.Fatalf("Failed to create temp token file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	initialToken := "token-version-1"
	if _, err := tmpFile.WriteString(initialToken); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	rs := NewRelayServerWithConfig("", tmpFile.Name())
	defer rs.Close()

	handler := rs.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// 1. Check with version 1 -> Must succeed
	req1, _ := http.NewRequest("GET", server.URL+"/api/v1/metrics", nil)
	req1.Header.Set("Authorization", "Bearer "+initialToken)
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK with initial token, got %v", resp1)
	}
	resp1.Body.Close()

	// 2. Rotate token in file dynamically (Simulating Infisical Secret Sync)
	rotatedToken := "token-version-2-rotated"
	if err := os.WriteFile(tmpFile.Name(), []byte(rotatedToken), 0600); err != nil {
		t.Fatalf("Failed to overwrite token file: %v", err)
	}

	// 3. Old token MUST now be rejected (401)
	reqOld, _ := http.NewRequest("GET", server.URL+"/api/v1/metrics", nil)
	reqOld.Header.Set("Authorization", "Bearer "+initialToken)
	respOld, err := http.DefaultClient.Do(reqOld)
	if err != nil || respOld.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized with old token, got %v", respOld)
	}
	respOld.Body.Close()

	// 4. New rotated token MUST be accepted immediately without server restart (Hot-Reload)
	reqNew, _ := http.NewRequest("GET", server.URL+"/api/v1/metrics", nil)
	reqNew.Header.Set("Authorization", "Bearer "+rotatedToken)
	respNew, err := http.DefaultClient.Do(reqNew)
	if err != nil || respNew.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK with rotated token, got %v", respNew)
	}
	respNew.Body.Close()
}

func TestRelayServerHeadMethodAndExistenceCheck(t *testing.T) {
	rs := NewRelayServer()
	defer rs.Close()

	handler := rs.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	code := "777-k0gni-88"
	payload := []byte("test payload for HEAD check")

	// 1. Before publishing -> HEAD returns 404
	exists, err := CheckRelayDropExists(server.URL, code)
	if err != nil {
		t.Fatalf("CheckRelayDropExists error: %v", err)
	}
	if exists {
		t.Errorf("Expected false before publish, got true")
	}

	// 2. Publish drop
	if err := PublishToRelay(server.URL, code, payload); err != nil {
		t.Fatalf("PublishToRelay error: %v", err)
	}

	// 3. After publishing -> HEAD returns 200 (without burning)
	exists, err = CheckRelayDropExists(server.URL, code)
	if err != nil {
		t.Fatalf("CheckRelayDropExists error: %v", err)
	}
	if !exists {
		t.Errorf("Expected true after publish, got false")
	}

	// 4. Fetch (Burn-After-Reading)
	data, err := FetchFromRelay(server.URL, code)
	if err != nil {
		t.Fatalf("FetchFromRelay error: %v", err)
	}
	if string(data) != string(payload) {
		t.Errorf("Payload mismatch: %s != %s", string(data), string(payload))
	}

	// 5. After fetch -> HEAD returns 404 (Burned)
	exists, err = CheckRelayDropExists(server.URL, code)
	if err != nil {
		t.Fatalf("CheckRelayDropExists error: %v", err)
	}
	if exists {
		t.Errorf("Expected false after burn, got true")
	}
}
