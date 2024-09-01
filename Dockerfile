FROM golang:1.22 AS builder

RUN mkdir "/go-drink"
WORKDIR /go-drink

COPY . /go-drink/

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o go-drink

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go-drink/go-drink /go-drink

ENTRYPOINT ["/go-drink"]
