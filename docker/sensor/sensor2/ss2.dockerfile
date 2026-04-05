FROM golang:1.25.3-alpine

WORKDIR /app

COPY go.mod ./
COPY docker/sensor/sensor2/sensor2.go ./main.go
RUN go build -o service main.go
CMD ["./service"]

