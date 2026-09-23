//go:build darwin

package platform

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type darwinBridge struct{}

func init() {
	SetBridge(&darwinBridge{})
}

func (d *darwinBridge) PlatformName() string {
	return "darwin"
}

func (d *darwinBridge) IsHeadless() bool {
	return false
}

func (d *darwinBridge) OpenURL(url string) error {
	cmd := exec.Command("open", url)
	return cmd.Start()
}

func (d *darwinBridge) LaunchTray(args []string, uiFallback func([]string) int) int {
	fs := flag.NewFlagSet("tray", flag.ExitOnError)
	install := fs.Bool("install", false, "Instala y registra Cogni en Aplicaciones")
	_ = fs.Parse(args)

	home, _ := os.UserHomeDir()
	appPath := filepath.Join(home, "Applications", "Cogni.app")

	if *install || !dirExists(appPath) {
		fmt.Println("Compilando e instalando Cogni.app nativo para macOS...")
		// Buscar script de compilacion
		buildScript := filepath.Join(home, ".cogni-src", "macos", "build.sh")
		if !fileExists(buildScript) {
			// Buscar relativo al directorio actual
			cwd, _ := os.Getwd()
			localScript := filepath.Join(cwd, "macos", "build.sh")
			if fileExists(localScript) {
				buildScript = localScript
			}
		}

		if fileExists(buildScript) {
			cmd := exec.Command("bash", buildScript, "--install")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error compilando Cogni: %v\n", err)
				return 1
			}
			return 0
		}
	}

	if dirExists(appPath) {
		fmt.Println("Iniciando Cogni en la barra de menús...")
		_ = exec.Command("open", appPath).Run()
		return 0
	}

	fmt.Println("Para compilar Cogni.app, ejecuta: cd macos && ./build.sh --install")
	return 0
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
