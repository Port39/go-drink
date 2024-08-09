FROM golang:1.22-alpine AS builder

RUN apk add --no-cache build-base

RUN mkdir "/go-drink"
WORKDIR /go-drink

COPY go.mod go.sum main.go data_transfer.go handlers.go /go-drink/
COPY items/*.go /go-drink/items/
COPY mailing/*.go /go-drink/mailing/
COPY mailing/templates/* /go-drink/mailing/templates/
COPY session/*.go /go-drink/session/
COPY transactions/*.go /go-drink/transactions/
COPY users/*.go /go-drink/users/

RUN CGO_ENABLED=1 go build -ldflags "-s -w" -trimpath -o /dist/go-drink
RUN ldd /dist/go-drink | tr -s [:blank:] '\n' | grep ^/ | xargs -I % install -D % /dist/%
RUN ln -s ld-musl-x86_64.so.1 /dist/lib/libc.musl-x86_64.so.1

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dist /

ENTRYPOINT ["/go-drink"]