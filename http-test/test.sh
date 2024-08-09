#!/bin/bash


TEST_GLOB="*.http"
OUTPUT="../test-reports"
GODRINK_HOST=localhost
GODRINK_PORT=8080
ENV_FILE="http-client.env.json"
ENV_NAME="dev"

__usage="
This script runs the http-tests against go drink using the intellij http client docker container
To that end, it tries to reach godrink at localhost 8080

Usage: $(basename $0) [OPTIONS]

Options:
  -t, --tests    <glob> --- A glob listing the .http files to test       --- Default: '$TEST_GLOB'
  --host         <glob> --- The host name at which to find godrink       --- Default: '$GODRINK_HOST'
  -p, --port     <glob> --- The port number under which to find godrink  --- Default: '$GODRINK_PORT'
  -r, --report   <path> --- Output directory for JUnit Test report       --- Default: '$OUTPUT'
  -e, --env-file <path> --- Path to the intellij http client env file    --- Default: '$ENV_FILE'
  --env-name     <path> --- Name of the intellij http client env         --- Default: '$ENV_NAME'
  -h, --help            --- Print this message
"

while [[ $# -gt 0 ]]; do
  case $1 in
    -h|"-?"|--help|help)
      HELP=1
      break
      ;;
    -t|--tests)
      TEST_GLOB="$2"
      shift # past argument
      shift # past value
      ;;
    --host)
      GODRINK_HOST="$2"
      shift # past argument
      shift # past value
      ;;
    -p|--port)
      GODRINK_PORT="$2"
      shift # past argument
      shift # past value
      ;;
    -r|--report)
      OUTPUT="$2"
      shift # past argument
      shift # past value
      ;;
    -e|--env-file)
      ENV_FILE="$2"
      shift # past argument
      shift # past value
      ;;
    --env-name)
      ENV_NAME="$2"
      shift # past argument
      shift # past value
      ;;
    -*|--*)
      echo "Unknown option $1"
      echo "$__usage"
      exit 1
      ;;
    *)
      echo "Unknown argument $1"
      exit 1
      ;;
  esac
done

[[ -n "$HELP" ]] && echo "$__usage" && exit 0

if [ ! -f $ENV_FILE ]; then
    echo "Environment file not found"
    echo "$__usage"
    exit 1
fi

set +e
while
(exec 6<>/dev/tcp/$GODRINK_HOST/$GODRINK_PORT) 2>/dev/null
[ $? == 1 ]
do echo Waiting for Go Drink; sleep 1; done
set -e

docker run --network="host" --rm -v $PWD:/workdir \
       jetbrains/intellij-http-client \
       --env "$ENV_NAME" \
       --env-file "$ENV_FILE" \
       --env-variables "godrink_url=$GODRINK_HOST:$GODRINK_PORT" \
       --report $OUTPUT \
        $TEST_GLOB
