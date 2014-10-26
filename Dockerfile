FROM golang:onbuild

# Default port for auth server
EXPOSE 8080

ENTRYPOINT ["/go/bin/app"]
