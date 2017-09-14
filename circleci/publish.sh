#!/bin/bash

set -eu

# Push go tgz
./godelw publish bintray --url https://api.bintray.com --subject palantir --repository releases --user "$BINTRAY_USERNAME" --password "$BINTRAY_PASSWORD" --publish --downloads-list duo-bot

version=$(./godelw project-version)

# Publish scratch docker container
docker login -e 'docker@palantir.com' -u "${DOCKERHUB_USERNAME}" -p "${DOCKERHUB_PASSWORD}" hub.docker.com
docker push palantirtechnologies/duo-bot:${version}
