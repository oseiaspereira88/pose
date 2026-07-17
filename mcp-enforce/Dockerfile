# mcp-enforce-sidecar: MCP enforcement reverse proxy (ADR-021). Applies the
# shared mcp-enforce policy gate + audit in front of a foreign MCP server
# (e.g. graphforge/mcp-server) on the composition network. Self-contained
# module (no external deps, no go.sum).
FROM golang:1.26.4-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /sidecar ./cmd/sidecar

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /sidecar /usr/local/bin/mcp-enforce-sidecar
ENV GF_SIDECAR_ADDR=:8770
EXPOSE 8770
ENTRYPOINT ["mcp-enforce-sidecar"]
