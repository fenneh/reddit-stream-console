FROM golang:1.24 AS builder

WORKDIR /app

COPY go/go.mod go/go.sum ./
RUN go mod download

COPY go/ .
RUN go build -o /app/bin/reddit-stream-console ./cmd/reddit-stream-console

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/bin/reddit-stream-console /app/reddit-stream-console
COPY config ./config
COPY .env.example ./

CMD ["./reddit-stream-console"]
