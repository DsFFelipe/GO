FROM golang:1.25.3-alpine

WORKDIR /app

COPY go.mod ./

COPY docker/cliente/cliente.go ./main.go
RUN go build -o service main.go
CMD ["./service"]