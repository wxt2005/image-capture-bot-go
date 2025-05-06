# Image capture bot

A telegram bot to capture images from Twitter, Pixiv and other image sites.

## Development

```bash
# fill config file
cp ./external/config-sample.yml ./external/config.yml
```

Use [air](https://github.com/cosmtrek/air) to auto compile and reload server in development.

```bash
# install air
go install github.com/air-verse/air@latest
# use air to enable hot reload
HOST=127.0.0.1 air
# endpoint is http://127.0.0.1:3000/
```

## Build Docker Image

```bash
docker build -t image-capture-bot-go:0.0.1 .
```

## Run Docker Container

```bash
docker run -it -p 3000:8080 -v /path/to/external:/go/image-capture-bot-go/external image-capture-bot-go:latest
```
