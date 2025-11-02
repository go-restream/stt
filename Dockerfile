FROM golang:1.23.4 AS builder
LABEL maintainer="streamASR team"

WORKDIR /app/

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Install build dependencies (for sherpa-onnx libraries)
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    wget \
    unzip \
    && ldconfig

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o streamASR .

# Copy sherpa-onnx shared libraries for VAD
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then \
        LIB_DIR="aarch64-unknown-linux-gnu"; \
    elif [ "$ARCH" = "armv7l" ]; then \
        LIB_DIR="arm-unknown-linux-gnueabihf"; \
    else \
        LIB_DIR="x86_64-unknown-linux-gnu"; \
    fi && \
    echo "Selected library directory: $LIB_DIR" && \
    cp /go/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@*/lib/${LIB_DIR}/*.so /usr/local/lib/

RUN ls /usr/local/lib/*.so

# RUN ARCH=$(uname -m) && \
#     if [ "$ARCH" = "aarch64" ]; then \
#         LIB_DIR="aarch64-unknown-linux-gnu"; \
#     elif [ "$ARCH" = "armv7l" ]; then \
#         LIB_DIR="arm-unknown-linux-gnueabihf"; \
#     else \
#         LIB_DIR="x86_64-unknown-linux-gnu"; \
#     fi && \
#     echo "Selected library directory: $LIB_DIR" && \
#     mkdir -p /tmp/sherpa-libs && \
#     find /go/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@* -name "*.so" -exec cp {} /usr/local/lib/\; || \
#     (echo "Sherpa-ONNX libraries not found in Go modules, they will be loaded at runtime" && exit 0)

FROM debian:stable-slim
LABEL maintainer="streamASR team"

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown
LABEL version="${VERSION}"
LABEL build-time="${BUILD_TIME}"
LABEL git-commit="${GIT_COMMIT}"

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    wget \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r streamasr && useradd -r -g streamasr streamasr

WORKDIR /app/

COPY --from=builder /app/streamASR .
COPY --from=builder /usr/local/lib/*.so /usr/local/lib/
RUN ls /usr/local/lib/*.so

RUN mkdir -p audio static model samples logs && \
    chown -R streamasr:streamasr /app/

USER streamasr

RUN chmod +x streamASR

USER root
RUN ldconfig || true
USER streamasr

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8088/v1/health || exit 1

EXPOSE 8088

CMD ["./streamASR"]