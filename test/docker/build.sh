#!/bin/bash

docker build ./tracker/ -t webtorrent-tracker
docker build ./client/ -t webtorrent-client