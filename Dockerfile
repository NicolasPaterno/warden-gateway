FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o gateway ./cmd/gateway/main.go

# ─────────────────────────────────────────────────────────────────────────────

FROM alpine:3.21

RUN addgroup -S warden && adduser -S gateway -G warden

WORKDIR /app

COPY --from=builder /app/gateway .

USER gateway

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
