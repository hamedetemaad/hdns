FROM golang:1.21.6 AS builder

RUN apt update && \
  apt install -y make bash git ca-certificates && \
  update-ca-certificates

WORKDIR /go/src/hdns-operator

COPY . .

RUN make build

FROM alpine:3.16.0

COPY --from=builder /go/src/hdns-operator/bin/hdns-operator /hdns-operator

CMD [ "/hdns-operator" ]