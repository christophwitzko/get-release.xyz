#!/bin/bash

set -e

# build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a --installsuffix cgo -ldflags="-s -w"

# download ca-certificates.crt
docker run -it --rm alpine /bin/sh -c "apk add --no-cache ca-certificates 2>&1 > /dev/null && cat /etc/ssl/certs/ca-certificates.crt" > ca-certificates.crt

# run semantic-release
semantic-release -vf
export VERSION=$(cat .version)

# docker build
export IMAGE_NAME="christophwitzko/grd-server"
export IMAGE_NAME_VERSION="$IMAGE_NAME:$VERSION"

docker build -t $IMAGE_NAME_VERSION .
docker tag $IMAGE_NAME_VERSION $IMAGE_NAME

# push to docker hub
docker login -u $DOCKER_USERNAME -p $DOCKER_PASSWORD
docker push $IMAGE_NAME_VERSION
docker push $IMAGE_NAME
