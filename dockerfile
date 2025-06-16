# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o obscure .

# Stage 2: Lightweight container
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/obscure .
ENTRYPOINT ["./obscure"]
