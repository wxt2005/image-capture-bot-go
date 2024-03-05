FROM ubuntu:focal

ARG GO_VERSION=1.21.3

ENV TZ=Asia/Tokyo
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN apt-get update \
    && apt-get install -y wget
RUN apt-get update \
    && apt-get install -y \
    wget build-essential pkg-config glib2.0-dev libexpat1-dev \
    libtiff5-dev libjpeg-turbo8-dev libgsf-1-dev libpng-dev libwebp-dev ffmpeg \
    libvips
RUN wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH $PATH:/usr/local/go/bin

RUN mkdir -p /go/image-capture-bot-go
WORKDIR /go/image-capture-bot-go
COPY . .

RUN go build -o app .

CMD ["./app"]
