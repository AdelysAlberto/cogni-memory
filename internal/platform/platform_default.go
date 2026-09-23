//go:build !darwin && !linux && !windows

package platform

import "fmt"

type defaultBridge struct{}

func init() {
	SetBridge(&defaultBridge{})
}

func (d *defaultBridge) PlatformName() string {
	return "unknown"
}

func (d *defaultBridge) IsHeadless() bool {
	return true
}

func (d *defaultBridge) OpenURL(url string) error {
	fmt.Printf("Abra el dashboard en: %s\n", url)
	return nil
}

func (d *defaultBridge) LaunchTray(args []string, uiFallback func([]string) int) int {
	if uiFallback != nil {
		return uiFallback(args)
	}
	return 0
}
