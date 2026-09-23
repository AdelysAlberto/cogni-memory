package platform

import (
	"testing"
)

func TestPlatformBridgeInitialized(t *testing.T) {
	b := Current()
	if b == nil {
		t.Fatal("Platform bridge was not initialized")
	}

	name := b.PlatformName()
	if name == "" {
		t.Fatal("Expected non-empty platform name")
	}

	// Test IsHeadless doesn't panic
	_ = IsHeadless()
}
