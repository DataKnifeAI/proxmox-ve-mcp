.PHONY: help build run test clean docker-build docker-run

help:
	@echo "Proxmox VE MCP Server - Available targets:"
	@echo "  build        - Build the binary"
	@echo "  run          - Run the server"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run in Docker container"

build:
	@echo "Building Proxmox VE MCP Server..."
	@mkdir -p bin
	go build -o bin/proxmox-ve-mcp ./cmd

run: build
	@echo "Running Proxmox VE MCP Server..."
	./bin/proxmox-ve-mcp

test:
	@echo "Running tests..."
	go test -v -cover ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

docker-build:
	@echo "Building Docker image..."
	docker build -t proxmox-ve-mcp:latest .

docker-run:
	@echo "Running in Docker..."
	docker-compose up
