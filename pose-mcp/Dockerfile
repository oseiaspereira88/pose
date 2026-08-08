# pose-mcp: read-only Streamable HTTP MCP server exposing POSE specs/workflows/rules
# (ADR-003). Serves a project mounted read-only at POSE_PROJECT_ROOT.
#
# Build context is the REPO ROOT: pose-mcp depends on the sibling module
# github.com/harne8/mcp-enforce via `replace => ../mcp-enforce` (ADR-021), which
# must be present in the build context. Build with:
#   docker build -f pose-mcp/Dockerfile .
FROM golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder
WORKDIR /app
# Neither module has external deps (no go.sum); copy both module trees.
COPY mcp-enforce/ ./mcp-enforce/
COPY pose-mcp/ ./pose-mcp/
WORKDIR /app/pose-mcp
RUN CGO_ENABLED=0 GOOS=linux go build -o /pose ./cmd/pose

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk --no-cache add ca-certificates git
COPY --from=builder /pose /usr/local/bin/pose
# Default to the port the Conductor and assembler expect for pose-mcp.
ENV POSE_MCP_ADDR=:8799
EXPOSE 8799
ENTRYPOINT ["pose", "serve-mcp"]
