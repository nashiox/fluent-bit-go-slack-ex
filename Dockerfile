FROM golang:1.14-buster AS builder
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOPATH=/go

ADD . /go/src/github.com/nashiox/fluent-bit-go-slack
WORKDIR /go/src/github.com/nashiox/fluent-bit-go-slack

RUN make build

FROM fluent/fluent-bit:1.5
LABEL Description="Fluent Bit Go Slack Extra" FluentBitVersion="1.5"

COPY --from=builder /go/src/github.com/nashiox/fluent-bit-go-slack/build/out_slack_ex.so /usr/lib/x86_64-linux-gnu/
COPY docker/fluent-bit-slack.conf /fluent-bit/etc/
EXPOSE 2020

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit-slack.conf", "-e", "/usr/lib/x86_64-linux-gnu/out_slack_ex.so"]
