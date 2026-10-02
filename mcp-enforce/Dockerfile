# mcp-enforce-sidecar: MCP enforcement reverse proxy (ADR-021). Applies the
# shared mcp-enforce policy gate + audit in front of a foreign MCP server
# (e.g. graphforge/mcp-server) on the composition network. Self-contained
# module (no external deps, no go.sum).
FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /sidecar ./cmd/sidecar

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
RUN apk --no-cache add ca-certificates
COPY --from=builder /sidecar /usr/local/bin/mcp-enforce-sidecar
ENV GF_SIDECAR_ADDR=:8770
EXPOSE 8770
ENTRYPOINT ["mcp-enforce-sidecar"]
