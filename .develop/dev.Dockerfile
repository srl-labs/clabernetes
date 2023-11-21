FROM --platform=linux/amd64 golang:1.21-bookworm

RUN set -x && apt-get update -y && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates wget jq procps curl vim inetutils-ping binutils && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /clabernetes

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

ENTRYPOINT ["sleep", "infinity"]
