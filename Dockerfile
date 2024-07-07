FROM golang:1.22 AS builder

RUN mkdir "/go-drink"
WORKDIR /go-drink

COPY go.mod go.sum main.go data_transfer.go handlers.go /go-drink/
COPY items/*.go /go-drink/items/
COPY session/*.go /go-drink/session/
COPY transactions/*.go /go-drink/transactions/
COPY users/*.go /go-drink/users/

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o go-drink

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go-drink/go-drink /go-drink

ENTRYPOINT ["/go-drink"]