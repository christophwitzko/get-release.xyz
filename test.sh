#!/bin/bash

set -e

curl localhost:5000/_usage | jq '.remaining'
curl localhost:5000/_/go
curl localhost:5000/semantic-release/linux/amd64
curl localhost:5000/semantic-release/go-semantic-release/linux/amd64
curl localhost:5000/go-semantic-release/go-semantic-release/linux/amd64
curl localhost:5000/go-semantic-release/go-semantic-release/linux/amd64/~1.6
