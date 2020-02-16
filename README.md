# Image capture bot

a telegram bot to capture images from Twitter, Pixiv and other social media;

## Devopment

```bash
# fill config file
cp ./external/config-sample.yml ./external/config.yml
```

Use [air](https://github.com/cosmtrek/air) to auto compile and reload server in development.

```bash
# install air
go get -u github.com/cosmtrek/air
# use air to enable hot reload
air
# endpoint is http://localhost:3000/
```

## Build Docker Image

```bash
docker build -t image-capture-bot-go:0.0.1 .
```

## Run Docker Container

```bash
docker run -it -p 3000:8080 -v /path/to/external:/go/image-capture-bot-go/external image-capture-bot-go:0.0.1
```
