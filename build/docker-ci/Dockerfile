FROM golang:1.8

WORKDIR /go/src/github.com/lezhenin/gotorrentclient

ADD . .

RUN go get -d -v ./...

ENTRYPOINT go test -timeout 1m -cover  ./pkg/torrent