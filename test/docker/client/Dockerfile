FROM node:latest

RUN npm install -g webtorrent-cli

WORKDIR /usr/torrent-test

ADD ./test/test_download ./download

RUN chmod -R +x .

ENTRYPOINT sh
