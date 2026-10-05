package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type MacOSBackend struct{}

func NewMacOSBackend() *MacOSBackend {
	return &MacOSBackend{}
}

func (b *MacOSBackend) IsInstalled() bool {
	home, _ := os.UserHomeDir()
	appPath := filepath.Join(home, "Applications", "Cogni.app")
	_, err := os.Stat(appPath)
	return err == nil
}

func (b *MacOSBackend) Install() error {
	home, _ := os.UserHomeDir()

	buildScript := filepath.Join(home, ".cogni-src", "macos", "build.sh")
	if _, err := os.Stat(buildScript); err != nil {
		cwd, _ := os.Getwd()
		localScript := filepath.Join(cwd, "macos", "build.sh")
		if _, err := os.Stat(localScript); err == nil {
			buildScript = localScript
		}
	}

	if _, err := os.Stat(buildScript); err != nil {
		return fmt.Errorf("script de compilación no encontrado: %s", buildScript)
	}

	fmt.Println("Compilando e instalando Cogni.app nativo para macOS...")
	cmd := exec.Command("bash", buildScript, "--install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *MacOSBackend) Run(ctx context.Context) error {
	home, _ := os.UserHomeDir()
	appPath := filepath.Join(home, "Applications", "Cogni.app")

	if !b.IsInstalled() {
		if err := b.Install(); err != nil {
			return fmt.Errorf("error instalando Cogni.app: %w", err)
		}
	}

	fmt.Println("Iniciando Cogni en la barra de menús...")
	cmd := exec.CommandContext(ctx, "open", appPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	if runtime.GOOS != "darwin" {
		return
	}
}