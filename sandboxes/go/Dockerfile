# go-runner
FROM golang:1.22.0-alpine

RUN adduser -D -u 1000 sandbox

WORKDIR /sandbox

USER sandbox

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["go run main.go"]
