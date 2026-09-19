package core

import (
	"strings"
	"testing"
)

func TestFormatTags(t *testing.T) {
	tags := FormatTags("auth, jwt, SECURITY, auth", "my-project")
	if !strings.Contains(tags, "my-project") {
		t.Errorf("Expected project tag to be included, got %s", tags)
	}
	if !strings.Contains(tags, "auth") {
		t.Errorf("Expected 'auth' tag, got %s", tags)
	}
	if !strings.Contains(tags, "security") {
		t.Errorf("Expected 'security' tag, got %s", tags)
	}
}

func TestEstimateTokens(t *testing.T) {
	tokens := EstimateTokens("12345678")
	if tokens != 2 {
		t.Errorf("Expected 2 tokens for 8 chars, got %d", tokens)
	}
}

func TestCleanProjectName(t *testing.T) {
	if cleanProjectName("/") != "" {
		t.Errorf("Expected '/' to be cleaned to empty string, got %s", cleanProjectName("/"))
	}
	if cleanProjectName(".") != "" {
		t.Errorf("Expected '.' to be cleaned to empty string, got %s", cleanProjectName("."))
	}
	if cleanProjectName("  ") != "" {
		t.Errorf("Expected whitespace to be cleaned to empty string, got %s", cleanProjectName("  "))
	}
	if cleanProjectName("my-app") != "my-app" {
		t.Errorf("Expected 'my-app', got %s", cleanProjectName("my-app"))
	}

	tags := FormatTags("auth", "/")
	if strings.Contains(tags, "/") {
		t.Errorf("FormatTags must not include '/' as a tag, got %s", tags)
	}
}
