#!/bin/bash

function retry {
  local retries=$1
  shift
  local count=0

  until "$@"; do
    exit=$?
    wait=$((2 ** count))
    count=$((count + 1))
    if [ "$count" -lt "$retries" ]; then
      sleep "$wait"
    else
      return "$exit"
    fi
  done

  return 0
}

export DEBIAN_FRONTEND=noninteractive
cd /tmp || exit 1
retry 5 sudo apt update || exit 1
retry 5 sudo apt install -y unzip awscli jq || exit 1
retry 5 sudo wget '${package_url}' || exit 1
retry 5 sudo unzip -o consul_*_linux_amd64.zip -d /usr/local/bin || exit 1

exit 0
