#############   builder   #############

FROM golang:1.13 AS builder

RUN curl -sfL "https://install.goreleaser.com/github.com/golangci/golangci-lint.sh" | sh -s -- -b $(go env GOPATH)/bin v1.21.0

WORKDIR /nsxt-lb-provider-manager/

ARG VERIFY=true

COPY . .

RUN make VERIFY=$VERIFY all


#############   nsxt-lb-provider-manager   #############

FROM alpine:latest AS nsxt-lb-provider-manager

RUN apk add --update bash curl

WORKDIR /

COPY --from=builder /go/bin/nsxt-lb-provider-manager /nsxt-lb-provider-manager
ENTRYPOINT ["/nsxt-lb-provider-manager"]
