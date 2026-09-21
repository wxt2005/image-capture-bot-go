ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22

# ---- build stage ----
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

# bimg needs cgo + libvips headers at compile time
RUN apk add --no-cache build-base pkgconf vips-dev

WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
# local-vendor is referenced by a replace directive in go.mod.
COPY go.mod go.sum ./
COPY local-vendor ./local-vendor
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /app .

# ---- runtime stage ----
FROM alpine:${ALPINE_VERSION}

ENV TZ=Asia/Tokyo

# vips: runtime libs for bimg; ffmpeg: called as a subprocess by ffmpeg-go
RUN apk add --no-cache vips ffmpeg tzdata ca-certificates

WORKDIR /go/image-capture-bot-go
COPY --from=builder /app ./app

CMD ["./app"]
