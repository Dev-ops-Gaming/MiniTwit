FROM golang:1.23

WORKDIR /app

COPY ./minitwit/go.mod ./minitwit/go.sum ./

RUN go mod download

COPY minitwit .

RUN go build -o main

EXPOSE 8080

CMD ["./main"]