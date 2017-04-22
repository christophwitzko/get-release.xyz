FROM scratch

ADD ca-certificates.crt /etc/ssl/certs/

COPY github-release-download /

EXPOSE 5000

ENTRYPOINT ["/github-release-download"]
