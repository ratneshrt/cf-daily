FROM golang:1.26.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o cf-daily \
    ./cmd/server

FROM alpine:3.22

# ca-certificates for the HTTPS calls to Telegram, Codeforces and GitHub;
# tzdata so time zone data is available if it is ever needed.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 cfdaily

WORKDIR /app

COPY --from=builder /app/cf-daily .

USER cfdaily

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O- http://127.0.0.1:8080/health || exit 1

CMD ["./cf-daily"]
