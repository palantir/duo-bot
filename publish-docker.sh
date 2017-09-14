#!/bin/bash

set -eu

# Publish scratch docker container
docker login -u "${DOCKERHUB_USERNAME}" -p "${DOCKERHUB_PASSWORD}" hub.docker.com
docker tag palantirtechnologies/duo-bot palantirtechnologies/duo-bot:$(./godelw project-version)
docker push palantirtechnologies/duo-bot
