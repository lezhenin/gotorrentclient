#!/bin/bash

echo ${ROLE}

if [[ ${ROLE} = 'SEEDER' ]];
then
    /usr/torrent-test/seeder.sh
elif [[ ${ROLE} = 'LEECHER' ]]
then
    /usr/torrent-test/leecher.sh
fi
