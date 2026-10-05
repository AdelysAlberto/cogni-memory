package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func handleTray(args []string) int {
	fs := flag.NewFlagSet("tray", flag.ExitOnError)
	install := fs.Bool("install", false, "Instala y registra Cogni en Aplicaciones (solo macOS)")
	_ = fs.Parse(args)

	backend := NewTrayBackend()

	if *install {
		if err := backend.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "Error instalando: %v\n", err)
			return 1
		}
		return 0
	}

	ctx := context.Background()
	if err := backend.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error ejecutando tray: %v\n", err)
		return 1
	}

	return 0
}
