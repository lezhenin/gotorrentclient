#!/bin/bash

docker network create net-torrent-test
docker run --network=net-torrent-test -itd --name=test-tracker webtorrent-tracker
docker run --network=net-torrent-test -itd --name=test-original-seeder --link test-tracker:tracker  webtorrent-client
docker run --network=net-torrent-test -itd --name=test-client-1 --link test-tracker:tracker --link test-original-seeder:seeder webtorrent-client
docker run --network=net-torrent-test -itd --name=test-client-2 --link test-tracker:tracker --link test-original-seeder:seeder webtorrent-client

# run seeder
docker exec -d test-original-seeder webtorrent download ./original/test_data.torrent -o ./original/ --keep-seeding -q

# run downloading
#docker exec -d test-client-1 webtorrent download ./original/test_data.torrent -o ./new/ --keep-seeding

# run downloading with blocked seeder
docker exec test-client-2 bash -c "getent hosts seeder | awk '{ print \$1 }' > blocklist.txt"

timeout --signal=SIGINT 10 \
docker exec test-client-2 webtorrent download ./original/test_data.torrent -o ./new/ -b ./blocklist.txt

# getent hosts seeder | awk '{ print $1 }'
# sudo docker exec -it
# webtorrent download ./original/test_data.torrent -o ./new/ --keep-seeding -b ./blocklist.txt
#

docker stop test-client-2
docker stop test-client-1
docker stop test-original-seeder
docker stop test-tracker

docker rm test-client-2
docker rm test-client-1
docker rm test-original-seeder
docker rm test-tracker