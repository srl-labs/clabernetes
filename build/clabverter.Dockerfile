FROM golang:1.21-bookworm as builder

ARG VERSION

WORKDIR /clabernetes

RUN mkdir build

COPY cmd/clabverter/main.go main.go

COPY . .

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

RUN CGO_ENABLED=0 \
    go build \
    -ldflags "-s -w -X github.com/srl-labs/clabernetes/constants.Version=${VERSION}" \
    -trimpath \
    -a \
    -o \
    build/clabverter \
    main.go

FROM debian:bookworm-slim

WORKDIR /clabverter
COPY --from=builder /clabernetes/build/clabverter /bin/clabverter

ENTRYPOINT ["/bin/clabverter"]