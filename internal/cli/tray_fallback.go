package cli

import (
	"context"
	"fmt"
	"runtime"
)

type FallbackBackend struct{}

func NewFallbackBackend() *FallbackBackend {
	return &FallbackBackend{}
}

func (b *FallbackBackend) IsInstalled() bool {
	return false
}

func (b *FallbackBackend) Install() error {
	return fmt.Errorf("system tray no disponible en %s", runtime.GOOS)
}

func (b *FallbackBackend) Run(ctx context.Context) error {
	return fmt.Errorf("system tray no disponible en %s. Use 'cogni ui' para abrir el dashboard", runtime.GOOS)
}

func init() {
	// No GOOS restriction - this is the fallback
}