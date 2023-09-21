FROM golang:1.21-bookworm as builder

ARG VERSION

WORKDIR /clabernetes

RUN mkdir certificates build

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

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /clabernetes
COPY --from=builder --chown=nonroot:nonroot /clabernetes/certificates /clabernetes/certificates
COPY --from=builder /clabernetes/build/manager .
USER nonroot:nonroot

ENTRYPOINT ["/clabernetes/manager", "run"]