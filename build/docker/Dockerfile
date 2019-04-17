FROM golang:1.8

WORKDIR /go/src/github.com/lezhenin/gotorrentclient

ADD . .

RUN go get -d -v ./...
RUN go install -v ./cmd/gotorrentcli/

ENTRYPOINT sh