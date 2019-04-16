FROM node:latest

RUN npm install -g webtorrent-cli

WORKDIR /usr/torrent-test

ADD test ./original

RUN chmod -R +x .

ENTRYPOINT sh
