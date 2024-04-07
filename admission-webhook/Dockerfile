FROM golang:1.21.6 AS builder
WORKDIR /app

COPY . .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hdns-admission-webhook .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/hdns-admission-webhook .
CMD ["./hdns-admission-webhook"]
