#!/usr/bin/env bash

echo "Restart $1 after $2 seconds..."
sleep $2
echo "Restart $1"
#docker kill $1
docker restart $1
