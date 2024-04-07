FROM golang:1.20 AS builder
WORKDIR /app

COPY . .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hdns .

FROM alpine:latest
RUN apk add iptables
WORKDIR /app
COPY --from=builder /app/hdns .
CMD ["./hdns"]
