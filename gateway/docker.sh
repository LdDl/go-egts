#!/bin/sh

echo "$DOCKER_REGISTRY_PASSWORD" | docker login "docker.io" -u "$DOCKER_REGISTRY_USERNAME" --password-stdin
DOCKER_BUILDKIT=1 docker build --ssh default=$HOME/.ssh/id_rsa -t egts_gateway -f gateway/Dockerfile .
docker tag egts_gateway "docker.io/$DOCKER_REGISTRY_USERNAME/egts_gateway"
docker push "docker.io/$DOCKER_REGISTRY_USERNAME/egts_gateway"
