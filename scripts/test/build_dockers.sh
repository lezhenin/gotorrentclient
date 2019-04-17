#!/bin/bash

# build docker with goTorrent cli
./scripts/build/build_cli_docker.sh

# build docker with goTorrent client for test
docker build -f ./test/docker/test-client/Dockerfile . -t gotorrent-client

# build docker with webtorrent tracker and client for test
docker build -f ./test/docker/tracker/Dockerfile . -t webtorrent-tracker
docker build -f ./test/docker//client/Dockerfile . -t webtorrent-client