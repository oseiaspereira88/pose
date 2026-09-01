# mcp-enforce-sidecar: MCP enforcement reverse proxy (ADR-021). Applies the
# shared mcp-enforce policy gate + audit in front of a foreign MCP server
# (e.g. graphforge/mcp-server) on the composition network. Self-contained
# module (no external deps, no go.sum).
FROM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /sidecar ./cmd/sidecar

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
RUN apk --no-cache add ca-certificates
COPY --from=builder /sidecar /usr/local/bin/mcp-enforce-sidecar
ENV GF_SIDECAR_ADDR=:8770
EXPOSE 8770
ENTRYPOINT ["mcp-enforce-sidecar"]
