FROM golang:1.21-bookworm as builder

ARG VERSION

WORKDIR /clabernetes

RUN mkdir build

COPY cmd/clabernetes/main.go main.go

COPY . .

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags "-s -w -X github.com/srl-labs/clabernetes/constants.Version=${VERSION}" \
    -trimpath \
    -a \
    -o \
    build/manager \
    main.go

FROM --platform=linux/amd64 debian:bookworm-slim

ARG DOCKER_VERSION="5:25.*"
ARG CONTAINERLAB_VERSION="0.48.*"
ARG NERDCTL_VERSION="1.7.2"

RUN apt-get update && \
    apt-get install -yq --no-install-recommends \
        ca-certificates \
        curl \
        wget \
        gnupg \
        lsb-release \
        vim \
        iproute2 \
        tcpdump \
        procps \
        openssh-client \
        inetutils-ping \
        traceroute

RUN echo "deb [trusted=yes] https://apt.fury.io/netdevops/ /" | \
    tee -a /etc/apt/sources.list.d/netdevops.list

RUN curl -fsSL https://download.docker.com/linux/debian/gpg | \
    gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

RUN echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian \
    $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

RUN apt-get update && \
    apt-get install -yq --no-install-recommends \
            containerlab=${CONTAINERLAB_VERSION} \
            docker-ce=${DOCKER_VERSION} \
            docker-ce-cli=${DOCKER_VERSION} && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /var/cache/apt/archive/*.deb

RUN wget -c https://github.com/containerd/nerdctl/releases/download/v${NERDCTL_VERSION}/nerdctl-${NERDCTL_VERSION}-linux-amd64.tar.gz -O - | tar -xz -C /usr/bin/ && rm /usr/bin/containerd-rootless*.sh

# https://github.com/docker/cli/issues/4807
RUN sed -i 's/ulimit -Hn/# ulimit -Hn/g' /etc/init.d/docker

# copy a basic but nicer than standard bashrc for the user
COPY build/launcher/.bashrc /root/.bashrc

# copy default ssh keys to the launcher image
# to make use of password-less ssh access
COPY build/launcher/default_id_rsa /root/.ssh/id_rsa
COPY build/launcher/default_id_rsa.pub /root/.ssh/id_rsa.pub
RUN chmod 600 /root/.ssh/id_rsa

# copy custom ssh config to enable easy ssh access from launcher
COPY build/launcher/ssh_config /etc/ssh/ssh_config

WORKDIR /clabernetes

RUN mkdir .node
RUN mkdir .image

COPY --from=builder /clabernetes/build/manager .
USER root

ENTRYPOINT ["/clabernetes/manager", "launch"]