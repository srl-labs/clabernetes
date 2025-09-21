FROM golang:1.24-bookworm AS builder

ARG VERSION

WORKDIR /clabernetes

RUN mkdir build

# certificates and subdirs need to be owned by root group for openshift reasons -- otherwise we
# get permission denied issues when the controller tries to create ca/client subdirs
RUN mkdir -p certificates/ca && \
    mkdir -p mkdir certificates/client && \
    mkdir -p mkdir certificates/webhook && \
    chgrp -R root /clabernetes/certificates && \
    chmod -R 0770 /clabernetes/certificates

COPY . .

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
    cmd/clabernetes/main.go

FROM --platform=linux/amd64 gcr.io/distroless/static-debian12:nonroot

WORKDIR /clabernetes
COPY --from=builder --chown=nonroot:nonroot /clabernetes/certificates /clabernetes/certificates
COPY --from=builder /clabernetes/build/manager .
USER nonroot:nonroot

ENTRYPOINT ["/clabernetes/manager", "run"]
