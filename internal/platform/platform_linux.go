//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
)

type linuxBridge struct{}

func init() {
	SetBridge(&linuxBridge{})
}

func (l *linuxBridge) PlatformName() string {
	return "linux"
}

// IsHeadless returns true if there is no active X11 or Wayland display session.
func (l *linuxBridge) IsHeadless() bool {
	return os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == ""
}

// OpenURL attempts to open the URL in the default Linux desktop browser with fallbacks.
func (l *linuxBridge) OpenURL(url string) error {
	if l.IsHeadless() {
		fmt.Fprintf(os.Stderr, "ℹ️ Sesión sin servidor gráfico ($DISPLAY no detectado).\n  Dashboard disponible en: %s\n", url)
		return nil
	}

	// 1. Try xdg-open
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err == nil {
		return nil
	}

	// 2. Try common browser fallbacks in Linux distributions
	fallbacks := []string{
		"sensible-browser",
		"x-www-browser",
		"firefox",
		"google-chrome",
		"chromium-browser",
		"chromium",
	}

	for _, browser := range fallbacks {
		if path, err := exec.LookPath(browser); err == nil {
			fallbackCmd := exec.Command(path, url)
			if err := fallbackCmd.Start(); err == nil {
				return nil
			}
		}
	}

	fmt.Fprintf(os.Stderr, "⚠️ No se pudo iniciar el navegador automáticamente.\n  Por favor abra la siguiente URL en su navegador: %s\n", url)
	return nil
}

// LaunchTray handles system tray / background UI for Linux environments.
func (l *linuxBridge) LaunchTray(args []string, uiFallback func([]string) int) int {
	fmt.Println("🐧 Cogni para Linux (Ubuntu/GNOME/KDE)")
	if l.IsHeadless() {
		fmt.Println("ℹ️ Entorno terminal / headless detectado. Iniciando servidor UI...")
	} else {
		fmt.Println("Iniciando servicio de Cogni e interfaz de usuario...")
	}

	if uiFallback != nil {
		return uiFallback(args)
	}
	return 0
}
