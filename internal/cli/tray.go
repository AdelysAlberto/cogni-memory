package cli

import (
	"github.com/AdelysAlberto/cogni/internal/platform"
)

func handleTray(args []string) int {
	return platform.LaunchTray(args, handleUI)
}

