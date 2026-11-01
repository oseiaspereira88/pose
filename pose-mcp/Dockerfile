# pose-mcp: read-only Streamable HTTP MCP server exposing POSE specs/workflows/rules
# (ADR-003). Serves a project mounted read-only at POSE_PROJECT_ROOT.
#
# Build context is the REPO ROOT: pose-mcp depends on the sibling module
# github.com/harne8/mcp-enforce via `replace => ../mcp-enforce` (ADR-021), which
# must be present in the build context. Build with:
#   docker build -f pose-mcp/Dockerfile .
FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder
WORKDIR /app
# Neither module has external deps (no go.sum); copy both module trees.
COPY mcp-enforce/ ./mcp-enforce/
COPY pose-mcp/ ./pose-mcp/
WORKDIR /app/pose-mcp
RUN CGO_ENABLED=0 GOOS=linux go build -o /pose ./cmd/pose

FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
RUN apk --no-cache add ca-certificates git
COPY --from=builder /pose /usr/local/bin/pose
# Default to the port the Conductor and assembler expect for pose-mcp.
ENV POSE_MCP_ADDR=:8799
EXPOSE 8799
ENTRYPOINT ["pose", "serve-mcp"]
