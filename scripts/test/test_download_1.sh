#!/bin/bash

# 1. run seeding client
# 2. run leeching client
# wait download to complete

TIMOUT_SECS=30
TEST_RETURN_CODE=0
TIMOUT_RETURN_CODE=0

echo "create docker network"
docker network create net-torrent-test

echo "run docker containers"
docker run --network=net-torrent-test -itd --name=test-tracker webtorrent-tracker
docker run --network=net-torrent-test -itd --name=test-seeder --link test-tracker:tracker  webtorrent-client
docker run --network=net-torrent-test -itd --name=test-leecher --link test-tracker:tracker --link test-original-seeder:seeder gotorrent-client

echo "run seeding client"
docker exec -d test-seeder \
webtorrent download ./download/test_data_docker.torrent -o ./download/ --keep-seeding -q

# wait announce
sleep 5

echo "run leeching client"
timeout --foreground --signal=SIGINT ${TIMOUT_SECS} \
docker exec test-leecher \
gotorrentcli -t ./download/test_data_docker.torrent -o ./output/ -v 3

TIMOUT_RETURN_CODE=$?
if [[ ${TIMOUT_RETURN_CODE} -ne 0 ]]; then
    echo "time is out: ${TIMOUT_SECS}"
    TEST_RETURN_CODE=1
fi

echo "stop docker containers"
docker stop test-leecher
docker stop test-seeder
docker stop test-tracker

echo "remove docker containers"
docker rm test-leecher
docker rm test-seeder
docker rm test-tracker

echo "remove docker network"
docker network rm net-torrent-test

exit ${TEST_RETURN_CODE}