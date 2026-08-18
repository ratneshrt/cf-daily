FROM golang:1.26.5-alphine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o cf-daily \
    ./cmd/server

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/cf-daily .

EXPOSE 8080

CMD ["./cf-daily"]