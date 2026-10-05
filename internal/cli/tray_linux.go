//go:build linux && cgo
// +build linux,cgo

package cli

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed assets/cogni-icon.png
var iconBytes []byte

//go:embed assets/cogni-icon.png
var iconFS embed.FS

func init() {
	SetLinuxBackendConstructor(func() TrayBackend { return NewLinuxBackend() })
}

type LinuxBackend struct {
	readyChan chan struct{}
}

func NewLinuxBackend() *LinuxBackend {
	return &LinuxBackend{
		readyChan: make(chan struct{}, 1),
	}
}

func (b *LinuxBackend) IsInstalled() bool {
	return true // Linux tray doesn't need installation
}

func (b *LinuxBackend) Install() error {
	return nil
}

func (b *LinuxBackend) Run(ctx context.Context) error {
	// Prepare icon
	if err := b.ensureIcon(); err != nil {
		return fmt.Errorf("error preparando icono: %w", err)
	}

	systray.SetIcon(iconBytes)
	systray.SetTitle("Cogni")
	systray.SetTooltip("Cogni - Cognitive Memory")

	mOpen := systray.AddMenuItem("Abrir Dashboard", "Abre el dashboard web de Cogni")
	mQuit := systray.AddMenuItem("Salir", "Cierra Cogni completamente")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				b.openDashboard()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			case <-ctx.Done():
				systray.Quit()
				return
			}
		}
	}()

	systray.Run(b.onReady, b.onExit)
	return nil
}

func (b *LinuxBackend) onReady() {
	close(b.readyChan)
}

func (b *LinuxBackend) onExit() {
}

func (b *LinuxBackend) openDashboard() {
	cmd := exec.Command("cogni", "ui", "--no-browser")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Start()
}

func (b *LinuxBackend) ensureIcon() error {
	// Try to read from embedded filesystem first
	data, err := iconFS.ReadFile("assets/cogni-icon.png")
	if err == nil && len(data) > 0 {
		iconBytes = data
		return nil
	}

	// Fallback: generate a simple icon programmatically
	iconBytes = generateDefaultIcon()
	return nil
}

func generateDefaultIcon() []byte {
	// Simple 16x16 PNG with Cogni colors (electric cyan on dark)
	// This is a minimal valid PNG
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0xF3, 0xFF, 0x61, 0x00, 0x00, 0x00,
		0x1E, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0xF8, 0x0F, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x05, 0x00, 0x0A, 0x49, 0x0C, 0x00, 0x1A,
		0x04, 0x82, 0x05, 0x3F, 0x0E, 0xC2, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
		0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}

func init() {
	if runtime.GOOS != "linux" {
		return
	}
}