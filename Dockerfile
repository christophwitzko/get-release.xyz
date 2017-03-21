FROM alpine:3.5

RUN apk add --no-cache ca-certificates

COPY github-release-download /

EXPOSE 5000

ENTRYPOINT ["/github-release-download"]
