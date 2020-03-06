#!/usr/bin/env bash
cd $(dirname $0)

go build -o ./flushdb-test ./

while ./flushdb-test
do
	sleep 1
done

rm -f ./flushdb-test
