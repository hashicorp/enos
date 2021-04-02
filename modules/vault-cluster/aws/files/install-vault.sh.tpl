#!/bin/bash

set -eux

export DEBIAN_FRONTEND=noninteractive
export fulllog="/tmp/vault_install.log"

exec > $fulllog 2>&1

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

# Wait for instances to boot
echo "Waiting for Vault instances to boot"
# We have seen some cases where our instance is up, we're connected and trying to install
# but something (cloud-init?) hasn't added the ec2 apt mirrors to the sources.list file yet.
# so we're going to sleep/retry until that happens
retry 10 grep ec2 /etc/apt/sources.list

cd /tmp
retry 5 sudo apt update

retry 5 wget -nv ${package_url}
retry 5 wget -nv https://releases.hashicorp.com/consul/1.9.3+ent/consul_1.9.3+ent_linux_amd64.zip

retry 5 sudo apt install -y unzip jq dnsmasq

retry 5 sudo unzip -o vault_*_linux_amd64.zip -d /usr/local/bin
retry 5 sudo unzip -o consul_*_linux_amd64.zip -d /usr/local/bin

# Give Vault the ability to use the mlock syscall without running the process as root
sudo setcap cap_ipc_lock=+ep /usr/local/bin/vault

# Create a unique, non-privileged system user to run Vault if it doesn't exist
id vault || sudo useradd -m --system --home /etc/vault.d --shell /bin/false vault

sudo cp /tmp/vault.service /etc/systemd/system