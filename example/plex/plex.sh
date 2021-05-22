#!/bin/bash
#
# This script is used by daemonless to invoke plex

docker \
  run \
  --name plex \
  --rm \
  --mount type=bind,source=/plex/database/,target=/config \
  --mount type=bind,source=/plex/transcode/,target=/transcode \
  --mount type=bind,source=/var/media/,target=/data \
  -p 32401:32400/tcp \
  -p 3005:3005/tcp \
  -p 8324:8324/tcp \
  -p 32469:32469/tcp \
  -p 1900:1900/udp \
  -p 32410:32410/udp \
  -p 32412:32412/udp \
  -p 32413:32413/udp \
  -p 32414:32414/udp \
  -e ADVERTISE_IP="http://plex.example.com:32400/" \
  -e TZ="America/Los_Angeles" -e PLEX_CLAIM=claim-xxxxxx \
  -e HOSTNAME="plex.example.com" \
  plexinc/pms-docker:plexpass
