package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/surrealwolf/proxmox-ve-mcp/internal/mcp"
	"github.com/surrealwolf/proxmox-ve-mcp/internal/proxmox"
)

func init() {
	// Load environment variables from .env file if it exists
	_ = godotenv.Load()

	// Configure logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		logrus.SetLevel(level)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Get configuration from environment
	baseURL := os.Getenv("PROXMOX_BASE_URL")
	if baseURL == "" {
		logrus.Fatal("PROXMOX_BASE_URL environment variable is required")
	}

	apiUser := os.Getenv("PROXMOX_API_USER")
	if apiUser == "" {
		logrus.Fatal("PROXMOX_API_USER environment variable is required")
	}

	apiToken := os.Getenv("PROXMOX_API_TOKEN")
	if apiToken == "" {
		logrus.Fatal("PROXMOX_API_TOKEN environment variable is required")
	}

	// Combine user and token into full API token format (user@realm!tokenid=secret)
	fullApiToken := fmt.Sprintf("%s!%s", apiUser, apiToken)

	// Check for SSL verification flag (default is to verify)
	skipSSLVerify := os.Getenv("PROXMOX_SKIP_SSL_VERIFY") == "true"
	if skipSSLVerify {
		logrus.Warn("SSL verification disabled - only use for self-signed certificates")
	}

	proxmoxClient := proxmox.NewClient(baseURL, fullApiToken, skipSSLVerify)

	// Initialize MCP server
	server := mcp.NewServer(proxmoxClient)

	// Determine transport mode
	transport := strings.ToLower(os.Getenv("MCP_TRANSPORT"))
	if transport == "" {
		transport = "stdio"
	}

	switch transport {
	case "http":
		httpAddr := os.Getenv("MCP_HTTP_ADDR")
		if httpAddr == "" {
			httpAddr = ":8000"
		}
		logrus.Infof("Starting Proxmox VE MCP Server on HTTP at %s", httpAddr)
		go func() {
			if err := server.ServeHTTP(httpAddr, ctx); err != nil {
				logrus.WithError(err).Fatal("HTTP Server error")
			}
		}()
	default:
		logrus.Info("Starting Proxmox VE MCP Server on stdio transport")
		go func() {
			if err := server.ServeStdio(ctx); err != nil {
				logrus.WithError(err).Fatal("Server error")
			}
		}()
	}

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down gracefully...")
	cancel()
	logrus.Info("Proxmox VE MCP Server stopped")
}
