# ── Stage 1: build
FROM golang:1.24.6 AS builder
WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники и собираем статически
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/app ./cmd/server

# ── Stage 2: runtime (легкий образ)
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /bin/app /app/app
COPY --from=builder /app/config/ /app/config/

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/app"]
