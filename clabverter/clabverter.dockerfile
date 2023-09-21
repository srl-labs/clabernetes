FROM golang:1.20-bookworm as builder

COPY . /workdir
WORKDIR /workdir

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o clabverter-bin ./cmd/clabverter/main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /workdir/clabverter-bin /bin/clabverter

USER nonroot:nonroot

ENTRYPOINT ["/bin/clabverter"]
