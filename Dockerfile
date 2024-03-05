FROM ubuntu:focal

ARG LIBVIPS_VERSION=8.13.3
ARG GO_VERSION=1.21.3

ENV TZ=Asia/Tokyo
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN apt-get update 
RUN apt-get remove libvips42
RUN apt-get install -y software-properties-common
RUN add-apt-repository ppa:lovell/cgif && apt-get update && apt-get install -y libcgif-dev
RUN apt-get install -y \
    build-essential \
    ninja-build \
    python3-pip \
    bc \
    wget
RUN pip3 install meson
RUN apt-get install -y \
    libfftw3-dev \
    libopenexr-dev \
    libgsf-1-dev \
    libglib2.0-dev \
    liborc-dev \
    libopenslide-dev \
    libmatio-dev \
    libwebp-dev \
    libjpeg-turbo8-dev \
    libexpat1-dev \
    libexif-dev \
    libtiff5-dev \
    libcfitsio-dev \
    libpoppler-glib-dev \
    librsvg2-dev \
    libpango1.0-dev \
    libopenjp2-7-dev \
    liblcms2-dev \
    libimagequant-dev
RUN wget -O- https://github.com/libvips/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.gz | tar xzC /tmp \
    && cd /tmp/vips-${LIBVIPS_VERSION} \
    && meson setup build --libdir=lib --buildtype=release -Dintrospection=false \
    && cd build \
    && meson compile \
    && meson test \
    && meson install \
    && ldconfig \
    && rm -rf /tmp/vips-${LIBVIPS_VERSION}
RUN wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH $PATH:/usr/local/go/bin

RUN mkdir -p /go/image-capture-bot-go
WORKDIR /go/image-capture-bot-go
COPY . .

RUN go build -o app .

CMD ["./app"]
