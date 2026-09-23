//go:build windows

package platform

import (
	"fmt"
	"os/exec"
)

type windowsBridge struct{}

func init() {
	SetBridge(&windowsBridge{})
}

func (w *windowsBridge) PlatformName() string {
	return "windows"
}

func (w *windowsBridge) IsHeadless() bool {
	return false
}

func (w *windowsBridge) OpenURL(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	return cmd.Start()
}

func (w *windowsBridge) LaunchTray(args []string, uiFallback func([]string) int) int {
	fmt.Println("Iniciando Cogni para Windows (lanzando UI)...")
	if uiFallback != nil {
		return uiFallback(args)
	}
	return 0
}
