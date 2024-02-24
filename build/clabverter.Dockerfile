FROM golang:1.22-bookworm as builder

ARG VERSION

WORKDIR /clabernetes

RUN mkdir build
RUN mkdir work

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

FROM --platform=linux/amd64 gcr.io/distroless/static-debian12:nonroot

WORKDIR /clabernetes
COPY --from=builder --chown=nonroot:nonroot /clabernetes/work /clabernetes/work
COPY --from=builder /clabernetes/build/clabverter .

WORKDIR /clabernetes/work

ENTRYPOINT ["/clabernetes/clabverter"]