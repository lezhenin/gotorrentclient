FROM golang:latest

WORKDIR /go/src/github.com/lezhenin/gotorrentclient

ADD . .

RUN go get -d -t -v ./...

ENTRYPOINT go test -timeout 5m -cover -v ./pkg/torrent
