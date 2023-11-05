# Build stage
FROM golang:1.19 as builder

WORKDIR /underwater-sensors

COPY . .

RUN go mod download

RUN go build -o sensor ./src/cmd

# Final stage
FROM alpine:3.14

WORKDIR /underwater-sensors

COPY --from=builder /underwater-sensors/sensor /underwater-sensors/sensor

EXPOSE 8080

CMD ["/underwater-sensors/sensor"]
