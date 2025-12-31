FROM golang:1.23.2-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxmox-ve-mcp ./cmd

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/proxmox-ve-mcp .

# Copy env example
COPY .env.example .

ENTRYPOINT ["./proxmox-ve-mcp"]
