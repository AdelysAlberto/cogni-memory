package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AdelysAlberto/cogni/internal/network"
)

var Version = "v2.4.0"

func main() {
	port := flag.Int("port", 8085, "Puerto para escuchar peticiones HTTP")
	host := flag.String("host", "0.0.0.0", "Host/IP de escucha")
	adminToken := flag.String("admin-token", os.Getenv("COGNI_RELAY_ADMIN_TOKEN"), "Token secreto de administración para métricas y vaciado")
	tokenFile := flag.String("token-file", os.Getenv("COGNI_RELAY_TOKEN_FILE"), "Ruta a archivo de token para recarga dinámica en caliente (ej. Infisical Agent)")
	ver := flag.Bool("version", false, "Muestra versión de cogni-relay")
	flag.Parse()

	if *ver {
		fmt.Printf("cogni-relay %s\n", Version)
		return
	}

	relay := network.NewRelayServerWithConfig(*adminToken, *tokenFile)
	defer relay.Close()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	server := &http.Server{
		Addr:         addr,
		Handler:      relay.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[cogni-relay %s] Servidor efímero iniciado en %s (RAM storage, E2EE, Burn-After-Reading)", Version, addr)
	log.Printf("   • Dashboard & Metrics: http://%s/", addr)
	log.Printf("   • Health check:        http://%s/health", addr)
	log.Printf("   • Drop endpoint:       http://%s/api/v1/drop", addr)
	if *tokenFile != "" {
		log.Printf("   • Dynamic Token File:  %s (Hot-reload activo via Infisical/disco)", *tokenFile)
	} else if *adminToken != "" {
		log.Printf("   • Admin Token:         Configurado via variable de entorno / flag")
	} else {
		log.Printf("   • Admin Token:         Desactivado (Acceso abierto / LAN)")
	}

	// Graceful shutdown handling
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error en servidor HTTP: %v", err)
		}
	}()

	<-stopCh
	log.Println("\nDeteniendo servidor cogni-relay...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error durante el cierre del servidor: %v", err)
	}

	log.Println("Servidor cogni-relay detenido correctamente.")
}
