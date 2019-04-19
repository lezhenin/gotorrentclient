FROM golang:latest

WORKDIR /go/src/github.com/lezhenin/gotorrentclient

ADD . .

RUN go get -d -t -v ./...

ENTRYPOINT go test -timeout 5m -cover -coverprofile=./test/coverage.out -v ./pkg/torrent && \
           go tool cover -func ./test/coverage.out
