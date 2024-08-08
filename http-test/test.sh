#!/bin/bash

set +e
while
(exec 6<>/dev/tcp/localhost/8080) 2>/dev/null
[ $? == 1 ]
do echo Waiting for Go Drink; sleep 1; done
set -e

docker run --network="host" --rm -v $PWD:/workdir jetbrains/intellij-http-client -e dev -v http-test/http-client.env.json -r test-reports  http-test/*.http
