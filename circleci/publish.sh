#!/bin/bash

set -eu

# Push go tgz
pushd "$GO_PROJECT_SRC_PATH"
./godelw publish bintray --url https://api.bintray.com --subject palantir --repository releases --user "$BINTRAY_USERNAME" --password "$BINTRAY_PASSWORD" --publish --downloads-list duo-bot
popd

# Publish scratch docker container
docker login -e 'docker@palantir.com' -u "${ARTIFACTORY_USERNAME}" -p "${ARTIFACTORY_PASSWORD}" docker.palantir.build
docker push palantir/duo-bot:${VERSION}
