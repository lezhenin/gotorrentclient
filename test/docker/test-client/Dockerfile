FROM gotorrent:latest

WORKDIR /usr/torrent-test

ADD ./test/test_download ./download

RUN chmod -R 777 .

ENTRYPOINT sh