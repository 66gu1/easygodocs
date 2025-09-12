# ── Stage 1: build
FROM golang:1.24.6 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/seedadmin ./cmd/seedadmin

# ── Stage 2: runtime
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /bin/server /app/server
COPY --from=builder /bin/seedadmin /app/seedadmin
COPY --from=builder /app/config/ /app/config/

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/server"]
