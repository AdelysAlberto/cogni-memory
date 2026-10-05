package cli

import (
	"context"
	"runtime"
)

type TrayBackend interface {
	Run(ctx context.Context) error
	Install() error
	IsInstalled() bool
}

// defaultNewLinuxBackend is the default constructor for Linux
var defaultNewLinuxBackend = func() TrayBackend { return NewFallbackBackend() }

func NewTrayBackend() TrayBackend {
	switch runtime.GOOS {
	case "darwin":
		return NewMacOSBackend()
	case "linux":
		return defaultNewLinuxBackend()
	default:
		return NewFallbackBackend()
	}
}

// SetLinuxBackendConstructor allows tests and build tags to override the Linux backend
func SetLinuxBackendConstructor(f func() TrayBackend) {
	defaultNewLinuxBackend = f
}