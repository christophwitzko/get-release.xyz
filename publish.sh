#!/bin/bash

set -e

# build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a --installsuffix cgo -ldflags="-s -w"

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
