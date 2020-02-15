FROM golang:1.13

RUN mkdir -p /go/image-capture-bot-go
WORKDIR /go/image-capture-bot-go
COPY . .

RUN go build -o app .

CMD ["./app"]
