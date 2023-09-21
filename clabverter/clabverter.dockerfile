FROM golang:1.20-bookworm as builder

COPY . /workdir
WORKDIR /workdir

ARG VERSION

RUN CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/srl-labs/clabernetes/constants.Version=${VERSION}" -trimpath -o clabverter-bin ./cmd/clabverter/main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /workdir/clabverter-bin /bin/clabverter

USER nonroot:nonroot

ENTRYPOINT ["/bin/clabverter"]
