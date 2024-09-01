FROM golang:1.22 AS builder

RUN mkdir "/go-drink"
WORKDIR /go-drink

COPY go.mod go.sum *.go /go-drink/
COPY items/*.go /go-drink/items/
COPY mailing/*.go /go-drink/mailing/
COPY mailing/templates/* /go-drink/mailing/templates/
COPY session/*.go /go-drink/session/
COPY transactions/*.go /go-drink/transactions/
COPY users/*.go /go-drink/users/
COPY domain_errors/*.go /go-drink/domain_errors/
COPY handlehttp/*.go /go-drink/handlehttp/
COPY handlehttp/content-type/*.go /go-drink/handlehttp/content-type/
COPY html-frontend/templates/*.gohtml /go-drink/html-frontend/templates/

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o go-drink

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go-drink/go-drink /go-drink

ENTRYPOINT ["/go-drink"]
