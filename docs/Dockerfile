FROM golang:1.23.4 AS builder
LABEL maintainer="XiaoYang"

WORKDIR /app/
COPY go.mod go.sum *.go configs/*.json VERSION scripts/  ./
RUN go mod download
COPY . .
RUN apt-get update && apt-get install -y libasound2-dev && ldconfig

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN go build -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" -o streamASR .
# sherpa-onnx shared libraries
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then \
        LIB_DIR="aarch64-unknown-linux-gnu"; \
    elif [ "$ARCH" = "armv7l" ]; then \
        LIB_DIR="arm-unknown-linux-gnueabihf"; \
    else \
        LIB_DIR="x86_64-unknown-linux-gnu"; \
    fi && \
    echo "Selected library directory: $LIB_DIR" && \
    cp /go/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@v1.12.2/lib/${LIB_DIR}/*.so /usr/local/lib/

FROM debian:stable-slim
LABEL maintainer="XiaoYang"

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown
LABEL version="${VERSION}"
LABEL build-time="${BUILD_TIME}"
LABEL git-commit="${GIT_COMMIT}"

COPY --from=builder /usr/local/lib/*.so /usr/local/lib/
COPY --from=builder /usr/local/lib/*.so.* /usr/local/lib/
RUN  ARCH=$(uname -m) && \
     cp /usr/local/lib/libasound.so.2 /usr/lib/${ARCH}-linux-gnu/ && \
     ldconfig
RUN  apt-get update && apt-get install -y ffmpeg && \
     apt clean && \
     rm -rf /var/lib/apt/lists/*
WORKDIR /app/
COPY models/kokoro-tts-multi-lang ./models/kokoro-tts-multi-lang
COPY models/fire-red-asr-large-zh_en-int8 ./models/fire-red-asr-large-zh_en-int8
COPY configs ./configs
COPY --from=builder /app/streamASR .
RUN chmod +x streamASR
EXPOSE 3000
CMD ["/app/streamASR"]