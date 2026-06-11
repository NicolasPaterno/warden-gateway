FROM golang:1.26-alpine AS builder

WORKDIR /src

# warden-auth is consumed as a public, versioned module (see go.mod), so this is a
# normal single-repo build: context is this directory, deps come from the module proxy.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /gateway ./cmd/gateway

# ─────────────────────────────────────────────────────────────────────────────

FROM alpine:3.21

RUN addgroup -S warden && adduser -S gateway -G warden

WORKDIR /app

COPY --from=builder /gateway .

USER gateway

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
