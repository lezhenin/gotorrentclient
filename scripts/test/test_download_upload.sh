#!/bin/bash

TIMOUT_SECS=30
TEST_RETURN_CODE=0
TIMOUT_RETURN_CODE=0

echo "create docker network"
docker network create net-torrent-test

echo "run docker containers"
docker run --network=net-torrent-test -itd --name=test-tracker -p 8000:8000 webtorrent-tracker
docker run --network=net-torrent-test -itd --name=test-original-seeder --link test-tracker:tracker  webtorrent-client
docker run --network=net-torrent-test -itd --name=test-client-1 --link test-tracker:tracker --link test-original-seeder:seeder gotorrent-client
docker run --network=net-torrent-test -itd --name=test-client-2 --link test-tracker:tracker --link test-original-seeder:seeder gotorrent-client

# run seeder
echo "run seeder"
docker exec -d test-original-seeder \
webtorrent download ./download/test_data_docker.torrent -o ./download/ --keep-seeding -q

# wait announce
sleep 5

# run leecher
echo "run client 1"
docker exec -i test-client-1 \
gotorrentcli -t ./download/test_data_docker.torrent -o ./output/ -s -v 3 &

sleep 15

echo "stop seeder"
docker stop test-original-seeder

echo "run client 2"
timeout --signal=SIGINT ${TIMOUT_SECS} \
docker exec test-client-2 \
gotorrentcli -t ./download/test_data_docker.torrent -o ./output/ -s -v 3 &

TIMOUT_RETURN_CODE=$?
if [[ ${TIMOUT_RETURN_CODE} -ne 0 ]]; then
    echo "time is out: ${TIMOUT_SECS}"
    TEST_RETURN_CODE=1
fi

echo "stop docker containers"
docker stop test-client-2
docker stop test-client-1
docker stop test-tracker

echo "remove docker containers"
docker rm test-client-2
docker rm test-client-1
docker rm test-original-seeder
docker rm test-tracker

echo "remove docker network"
docker network rm net-torrent-test

exit ${TEST_RETURN_CODE}