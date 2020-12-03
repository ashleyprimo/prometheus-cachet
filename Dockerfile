# Building
# --------
FROM golang:1.14-alpine as builder

ARG SRC=$GOPATH/src/github.com/ashleyprimo/prometheus-cachet

COPY . $SRC
WORKDIR $SRC

RUN apk add git && \
go get && \
go build --ldflags '-extldflags "-static"' -o bin/prometheus-cachet

# Deployment
# ----------
FROM alpine:3.12

ARG REPO=/go/src/github.com/ashleyprimo/prometheus-cachet

COPY --from=builder $REPO/bin/* /usr/local/bin/

RUN apk add --no-cache ca-certificates

ENTRYPOINT [ "prometheus-cachet" ]
