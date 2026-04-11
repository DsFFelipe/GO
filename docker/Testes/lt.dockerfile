FROM golang:1.25.3-alpine

WORKDIR /app

COPY go.mod ./
COPY loadtest.go ./main.go
RUN go build -o loadtester main.go

CMD ["./loadtester"]