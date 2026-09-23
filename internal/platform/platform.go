package platform

// Bridge defines the contract for OS-specific desktop and system integrations.
type Bridge interface {
	OpenURL(url string) error
	LaunchTray(args []string, uiFallback func([]string) int) int
	IsHeadless() bool
	PlatformName() string
}

var currentBridge Bridge

// SetBridge registers the active platform bridge (called by platform-specific init)
func SetBridge(b Bridge) {
	currentBridge = b
}

// Current returns the platform bridge for the current operating system.
func Current() Bridge {
	return currentBridge
}

// OpenURL opens the specified URL in the system browser with robust error reporting.
func OpenURL(url string) error {
	if currentBridge == nil {
		return nil
	}
	return currentBridge.OpenURL(url)
}

// LaunchTray launches the OS-specific system tray/menu bar or falls back to UI.
func LaunchTray(args []string, uiFallback func([]string) int) int {
	if currentBridge == nil {
		if uiFallback != nil {
			return uiFallback(args)
		}
		return 0
	}
	return currentBridge.LaunchTray(args, uiFallback)
}

// IsHeadless checks if running in an environment without graphical display.
func IsHeadless() bool {
	if currentBridge == nil {
		return false
	}
	return currentBridge.IsHeadless()
}
