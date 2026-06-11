FROM golang:1.26-alpine AS builder

# Multi-repo build: the build context must be the parent directory holding both
# warden-gateway and warden-auth, because go.mod has
# `replace github.com/NicolasPaterno/warden-auth => ../warden-auth` (sibling repo).
# Build with context at the parent, e.g.:
#   docker build -f warden-gateway/Dockerfile -t warden-gateway ..
WORKDIR /src

COPY warden-auth/go.mod warden-auth/go.sum ./warden-auth/
COPY warden-gateway/go.mod warden-gateway/go.sum ./warden-gateway/
WORKDIR /src/warden-gateway
RUN go mod download

WORKDIR /src
COPY warden-auth/ ./warden-auth/
COPY warden-gateway/ ./warden-gateway/

WORKDIR /src/warden-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /gateway ./cmd/gateway

# ─────────────────────────────────────────────────────────────────────────────

FROM alpine:3.21

RUN addgroup -S warden && adduser -S gateway -G warden

WORKDIR /app

COPY --from=builder /gateway .

USER gateway

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
