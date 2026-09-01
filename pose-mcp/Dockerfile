# pose-mcp: read-only Streamable HTTP MCP server exposing POSE specs/workflows/rules
# (ADR-003). Serves a project mounted read-only at POSE_PROJECT_ROOT.
#
# Build context is the REPO ROOT: pose-mcp depends on the sibling module
# github.com/harne8/mcp-enforce via `replace => ../mcp-enforce` (ADR-021), which
# must be present in the build context. Build with:
#   docker build -f pose-mcp/Dockerfile .
FROM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder
WORKDIR /app
# Neither module has external deps (no go.sum); copy both module trees.
COPY mcp-enforce/ ./mcp-enforce/
COPY pose-mcp/ ./pose-mcp/
WORKDIR /app/pose-mcp
RUN CGO_ENABLED=0 GOOS=linux go build -o /pose ./cmd/pose

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
RUN apk --no-cache add ca-certificates git
COPY --from=builder /pose /usr/local/bin/pose
# Default to the port the Conductor and assembler expect for pose-mcp.
ENV POSE_MCP_ADDR=:8799
EXPOSE 8799
ENTRYPOINT ["pose", "serve-mcp"]
