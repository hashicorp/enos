#!/bin/bash

set -x

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

export initfile="/tmp/vault-init"
export logfile="/tmp/vault.log"
export fulllog="/tmp/vault_config.log"

exec > $fulllog 2>&1

# Copy the vault config for systemd
sudo cp /tmp/server.hcl /etc/vault.d/vault.hcl
sudo chmod 640 /etc/vault.d/vault.hcl
sudo chown -R vault:vault /etc/vault.d

sudo systemctl --now enable vault

export VAULT_ADDR=http://localhost:8200

sudo systemctl status vault > /tmp/vault_sysctl_status.out
vcount=0

until vault status
do
  vcount=$((vcount + 1))
  retries=6
  vault status
  status_code=$?
  case $status_code in
    0)
      # Alive, unsealed
      echo "The cluster is already Initialized at $(vault status | grep Node)" >> $logfile
      exit 0
    ;;
    1)
      # Connection error, try 5 times with backup and then restart
      echo "Unable to connect to vault, retrying" >> $logfile
      # We're catching the vault-is-starting but not listening condition
      retry 5 nc -z 127.0.0.1 8200
      continue
    ;;
    2)
      # Vault Service is running, but Sealed
      if [ "$(vault status | grep Initialized | awk '{print $2}')" == "false" ]
      then
        # Initialize the cluster and store root token in json format
        vault operator init -format json > $initfile 2>>/tmp/vault-init.err
        vault status >> /tmp/vault_status.orig 2>>/tmp/vault-init.err

        if [ "$(vault status | grep Sealed |awk '{print $2}')" == "true" ]
        then
          # If vault initializes, but doesn't auto unseal itself, restart
          echo "Vault initialized, didn't unseal" >> $logfile
          sudo systemctl restart vault
          continue
        fi
        export copy=$(curl -s 169.254.169.254/1.0/meta-data/local-ipv4)
        export VAULT_TOKEN=$(cat $initfile | jq -r '.root_token')
        sudo mv $initfile /etc/vault.d/vault-init.$copy
        # Save the root token incase the file gets overwritten
        echo $VAULT_TOKEN |sudo tee /etc/vault.d/tokens.$copy
        retry 5 vault write /sys/license text=${vault_license}
        retry 5 vault secrets enable -path="secret" kv
        retry 5 vault kv put secret/test test=meow
      else
        echo "The cluster is already Initialized at $(vault status |grep Node)" >> $logfile
        vault status | grep Sealed | grep false &>> $logfile|| sudo systemctl restart vault 
      fi
    ;;
  esac
  if [ "$vcount" -lt "$retries" ]; then
    sleep 5
  else
    echo "Vault status continued to return unexpected status" >> $logfile
    exit 1
  fi
done
