package mcp

import (
	"slices"
	"testing"
)

func TestMCPMemoryCategorySchemas(t *testing.T) {
	expected := []string{"bugfix", "architecture", "refactor", "decision", "discovery", "config", "pattern", "preference", "general"}
	server := NewServer("test-categories")
	checked := 0
	for _, tool := range server.getToolsList() {
		if tool.Name != "cogni_save" && tool.Name != "cogni_search" {
			continue
		}
		checked++
		t.Run(tool.Name, func(t *testing.T) {
			category, exists := tool.InputSchema.Properties["category"]
			if !exists {
				t.Fatal("Missing category property")
			}
			if !slices.Equal(category.Enum, expected) {
				t.Errorf("Expected category enum %v, got %v", expected, category.Enum)
			}
		})
	}
	if checked != 2 {
		t.Fatalf("Expected save and search category schemas, checked %d", checked)
	}
}
