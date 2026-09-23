package network

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/storage"
)

func TestPairCodeFormat(t *testing.T) {
	for i := 0; i < 50; i++ {
		code := GeneratePairCode()
		if !ValidatePairCode(code) {
			t.Fatalf("Generated code '%s' failed validation", code)
		}
	}

	// Negative tests
	if ValidatePairCode("12-c0gn1-42") {
		t.Errorf("Expected false for 2-digit first slot")
	}
	if ValidatePairCode("381-c0gn1-4") {
		t.Errorf("Expected false for 1-digit third slot")
	}
	if ValidatePairCode("381c0gn142") {
		t.Errorf("Expected false for missing hyphens")
	}
}

func TestCryptoRoundtrip(t *testing.T) {
	code := "381-c0gn1-42"
	original := []byte("Decision de arquitectura: usar Go estándar para networking multiplataforma.")

	encrypted, err := Encrypt(original, code)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Must not match plaintext
	if string(encrypted) == string(original) {
		t.Fatalf("Encrypted payload matches plaintext")
	}

	// Decrypt with correct code
	decrypted, err := Decrypt(encrypted, code)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(original) {
		t.Errorf("Expected '%s', got '%s'", string(original), string(decrypted))
	}

	// Decrypt with wrong code must fail
	_, errWrong := Decrypt(encrypted, "999-k0gni-11")
	if errWrong == nil {
		t.Fatalf("Expected decryption failure with wrong code, but succeeded")
	}
}

func TestMergeSafeForkOnConflict(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-network-merge-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "merge_test.db")
	s, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer s.Close()

	// 1. Local user has existing memory
	localMem := &core.Memory{
		ProjectName:      "viasera",
		Category:         "architecture",
		Title:            "Auth Strategy Local",
		TopicKey:         "arch/auth/jwt",
		SummarySignature: "Local developer notes on JWT.",
		Tags:             "jwt,auth,local",
	}
	_, err = s.SaveMemory(localMem)
	if err != nil {
		t.Fatalf("Failed to save local memory: %v", err)
	}

	// 2. Incoming memory from peer with DIFFERENT content
	incoming := []core.Memory{
		{
			ProjectName:      "viasera",
			Category:         "architecture",
			Title:            "Auth Strategy Peer",
			TopicKey:         "arch/auth/jwt",
			SummarySignature: "Remote peer modified notes with new requirements.",
			Tags:             "jwt,auth,remote",
		},
		{
			ProjectName:      "viasera",
			Category:         "database",
			Title:            "Database Indexing Strategy",
			TopicKey:         "arch/db/indexes",
			SummarySignature: "Brand new memory from peer.",
			Tags:             "db,indexes",
		},
	}

	res, err := MergeMemories(s, incoming, "alice")
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if res.Inserted != 1 {
		t.Errorf("Expected 1 inserted, got %d", res.Inserted)
	}
	if res.Conflicts != 1 {
		t.Errorf("Expected 1 conflict, got %d", res.Conflicts)
	}

	// Invariant Check: Local user memory must remain intact!
	intactLocal, err := s.GetMemoryByTopicKey("viasera", "arch/auth/jwt")
	if err != nil || intactLocal == nil {
		t.Fatalf("Local memory was lost or corrupted: %v", err)
	}
	if intactLocal.Title != "Auth Strategy Local" {
		t.Errorf("Local memory was overwritten! Title is %s", intactLocal.Title)
	}

	// Invariant Check: Forked peer memory must exist safely!
	forkedPeer, err := s.GetMemoryByTopicKey("viasera", "arch/auth/jwt@peer-alice")
	if err != nil || forkedPeer == nil {
		t.Fatalf("Forked peer memory was not created: %v", err)
	}
	if forkedPeer.Title != "Auth Strategy Peer (peer: alice)" {
		t.Errorf("Unexpected forked title: %s", forkedPeer.Title)
	}
}

func TestP2PEphemeralSessionAndSelfDestruct(t *testing.T) {
	// 1. Prepare sender memories
	memories := []core.Memory{
		{
			ProjectName:      "cloud-proj",
			Category:         "infra",
			Title:            "Docker Swarm Setup",
			TopicKey:         "infra/swarm",
			SummarySignature: "Docker swarm setup notes for cross-platform team.",
			Tags:             "docker,swarm",
		},
	}

	// 2. Start ephemeral session
	session, err := StartShareSession("cloud-proj", memories, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}
	defer session.Close()

	if !ValidatePairCode(session.Code) {
		t.Fatalf("Invalid pair code: %s", session.Code)
	}

	// 3. Prepare receiver storage
	tmpDir, err := os.MkdirTemp("", "cogni-receiver-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "receiver.db")
	receiverStorage, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init receiver storage: %v", err)
	}
	defer receiverStorage.Close()

	// 4. Connect receiver to sender using direct address
	directAddr := fmt.Sprintf("127.0.0.1:%d", session.Port)
	resp, err := SyncFromPeer(receiverStorage, session.Code, directAddr)
	if err != nil {
		t.Fatalf("SyncFromPeer failed: %v", err)
	}

	if resp.Inserted != 1 {
		t.Errorf("Expected 1 memory inserted, got %d", resp.Inserted)
	}

	// Verify imported memory exists in receiver
	imported, err := receiverStorage.GetMemoryByTopicKey("cloud-proj", "infra/swarm")
	if err != nil || imported == nil {
		t.Fatalf("Imported memory not found in receiver storage: %v", err)
	}
	if imported.Title != "Docker Swarm Setup" {
		t.Errorf("Expected title 'Docker Swarm Setup', got %s", imported.Title)
	}

	// 5. Invariant: Session must self-destruct after transfer!
	// Give a small grace period for the background self-destruct
	time.Sleep(800 * time.Millisecond)

	// Attempting a second sync must FAIL because the session died
	_, secondErr := SyncFromPeer(receiverStorage, session.Code, directAddr)
	if secondErr == nil {
		t.Fatalf("Expected second sync to fail because session must self-destruct, but it succeeded")
	}
}

