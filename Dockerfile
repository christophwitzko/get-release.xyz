FROM golang:1.16 AS builder
WORKDIR /app

COPY go.* ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build -ldflags="-extldflags '-static' -s -w" ./cmd/get-release-server/

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/get-release-server .

EXPOSE 5000
CMD ["/get-release-server"]
