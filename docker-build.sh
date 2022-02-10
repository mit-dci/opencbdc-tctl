#!/bin/bash
set -e
export GIT_DATE=$(git log -1 --format=%cd --date=format:"%Y%m%d")
export GIT_COMMIT=$(git rev-list -1 HEAD)
docker build . --build-arg GIT_DATE=$GIT_DATE --build-arg GIT_COMMIT=$GIT_COMMIT -f Dockerfile.agent -t opencbdc-tct-agent
docker build . --build-arg GIT_DATE=$GIT_DATE --build-arg GIT_COMMIT=$GIT_COMMIT -f Dockerfile.coordinator -t opencbdc-tct-controller

