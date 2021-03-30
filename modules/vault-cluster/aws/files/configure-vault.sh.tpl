#!/bin/bash

set -eux

export initfile="/tmp/vault-init"
export logfile="/tmp/vault.log"
export fulllog="/tmp/vault_config.log"

exec > $fulllog 2>&1

# Copy the vault config for systemd
sudo cp /tmp/server.hcl /etc/vault.d/vault.hcl
sudo chmod 640 /etc/vault.d/vault.hcl
sudo chown -R vault:vault /etc/vault.d

# Sleep a random value to prevent trying to start/init simultaneously
export wait="$[( $RANDOM % 50 )]"
echo "Sleeping $wait seconds before init" >> $logfile
sleep "$wait"s

sudo systemctl enable vault
sudo systemctl start vault
export VAULT_ADDR=http://localhost:8200

#  Give Vault service some time to startup
sleep 20
sudo systemctl status vault > /tmp/vault_sysctl_status.out

# Init the cluster regardless of state, it fails if already set, store 
# root token in json format on primary
if [ "$(vault status |grep Initialized |awk '{print $2}')" == "false" ]
then
  vault operator init -format json > $initfile 2>>/tmp/vault-init.err
  vault status > /tmp/vault_status.orig 2>>/tmp/vault-init.err
  sleep 20
  vault status | grep Sealed | grep false || sudo systemctl restart vault &>> $logfile
else
  echo "The cluster is already Initialized at $(vault status |grep Node)" >> $logfile
  sleep 20
  vault status | grep Sealed | grep false || sudo systemctl restart vault &>> $logfile
fi

# if the primary node, do things with the root token
if [ -s $initfile ]
then
  export copy=$(curl -s 169.254.169.254/1.0/meta-data/local-ipv4)
  export VAULT_TOKEN=$(cat $initfile | jq -r '.root_token')
  sudo mv $initfile /etc/vault.d/vault-init.$copy
  # Save the root token incase the file gets overwritten
  echo $VAULT_TOKEN |sudo tee /etc/vault.d/tokens.$copy
  vault write /sys/license text=${vault_license}
  vault secrets enable -path="secret" kv
  vault kv put secret/test test=meow
else
  export VAULT_ADDR=http://localhost:8200
  vault status | grep Sealed | grep false || sudo systemctl restart vault &>> $logfile
fi