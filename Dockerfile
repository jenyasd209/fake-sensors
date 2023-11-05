FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o sensor ./src/cmd/main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/sensor /app/sensor

CMD ["./sensor"]