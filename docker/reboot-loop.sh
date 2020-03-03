#!/usr/bin/env bash

while [ 1 ]; do
    n=$RANDOM
    let "n%=120"
    ./restart-after.sh node1 $n &
    echo "Start..."
    ./start.sh

    echo "Check node down"
    present=`docker ps|grep node1`
    if [ "$present" == "" ]; then
	./start.sh
	present=`docker ps|grep node1`
	if [ "$present" == "" ]; then
	    ./stop.sh
	    exit
	fi
    fi
done
