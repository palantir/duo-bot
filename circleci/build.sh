#!/bin/bash

set -eux

./godelw verify --apply=false --junit-output="${ARTIFACT_STORE}/tests.xml"
./godelw dist

# We need to trust duosecurity.com, so let's just use system CA certs
# The machine pre section makes sure we've got the latest from apt
cp /etc/ssl/certs/ca-certificates.crt .

docker build \
    -t palantir/duo-bot:${VERSION} \
    --build-arg VERSION=$VERSION \
    -f Dockerfile \
    .

# Otherwise our version tags will say we're dirty
rm -f ca-certificates.crt
