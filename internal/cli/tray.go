package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func handleTray(args []string) int {
	fs := flag.NewFlagSet("tray", flag.ExitOnError)
	install := fs.Bool("install", false, "Instala y registra CogniBar en Aplicaciones")
	_ = fs.Parse(args)

	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		appPath := filepath.Join(home, "Applications", "CogniBar.app")
		
		if *install || !dirExists(appPath) {
			fmt.Println("🔨 Compilando e instalando CogniBar.app nativo para macOS...")
			// Buscar script de compilación
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
					fmt.Fprintf(os.Stderr, "Error compilando CogniBar: %v\n", err)
					return 1
				}
				return 0
			}
		}

		if dirExists(appPath) {
			fmt.Println("🚀 Iniciando CogniBar en la barra de menús...")
			_ = exec.Command("open", appPath).Run()
			return 0
		}

		fmt.Println("💡 Para compilar CogniBar.app, ejecuta: cd macos && ./build.sh --install")
		return 0
	}

	// Linux / Windows: Lanzar dashboard UI en modo daemon/ventana
	fmt.Printf("🌐 Iniciando Cogni Tray para %s (lanzando UI)...\n", runtime.GOOS)
	return handleUI(args)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
