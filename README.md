# Image capture bot

a telegram bot to capture images from Twitter, Pixiv and other social media;

## Dev

```bash
cp ./external/config-sample.yml ./external/config.yml
# fill blanks
```

Use [gin](https://github.com/codegangsta/gin) to auto compile and reload server while developing.

```bash
gin run main.go
# http://localhost:3000/
```

## Build Docker Image

```bash
docker build -t image_capture_bot_go:0.0.1 .
```

## Run Docker Container

```bash
docker run -it -p 3000:8080 -v /path/to/external:/go/src/github.com/wxt2005/image_capture_bot_go/external image_capture_bot_go:0.0.1
```
