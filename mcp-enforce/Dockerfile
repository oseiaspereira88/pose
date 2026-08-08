# mcp-enforce-sidecar: MCP enforcement reverse proxy (ADR-021). Applies the
# shared mcp-enforce policy gate + audit in front of a foreign MCP server
# (e.g. graphforge/mcp-server) on the composition network. Self-contained
# module (no external deps, no go.sum).
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder
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
