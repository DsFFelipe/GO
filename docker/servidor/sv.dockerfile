FROM golang:1.25.3-alpine

WORKDIR /app

COPY go.mod ./

COPY servidor.go ./main.go

EXPOSE 8080/udp
EXPOSE 8080/tcp
EXPOSE 8082/tcp

RUN go build -o service main.go
CMD ["./service"]