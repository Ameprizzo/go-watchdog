# Stage 1: Build the binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o watchdog cmd/watchdog/main.go

# Stage 2: Final lightweight image
FROM alpine:latest
WORKDIR /root/
# Copy the binary from the builder stage
COPY --from=builder /app/watchdog .
# Copy the config and web assets (they are needed at runtime)
COPY --from=builder /app/config.json .
COPY --from=builder /app/web ./web

EXPOSE 8080
CMD ["./watchdog"]