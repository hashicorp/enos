#!/bin/bash
cd /tmp
apt update
apt install -y unzip awscli jq dnsmasq
wget -nv ${package_url}
wget -nv https://releases.hashicorp.com/consul/1.9.3+ent/consul_1.9.3+ent_linux_amd64.zip

unzip vault_*_linux_amd64.zip -d /usr/local/bin
unzip consul_*_linux_amd64.zip -d /usr/local/bin

echo "# Enable forward lookup of the 'consul' domain:
server=/consul/127.0.0.1#8600
rev-server=10.0.0.0/8,127.0.0.1#8600" >> /etc/dnsmasq.d/consul.conf
service dnsmasq restart

#IPs of consul nodes, we could also use AWS lookup
IPS="${consul_ips}"
nohup consul agent -join $${IPS// / -join } -data-dir /tmp > consul.out&

echo 'api_addr = "http://localhost:8200"
cluster_addr = "http://localhost:8201"
ui = true

storage "consul" {
  address = "127.0.0.1:8500"
  path    = "vault"
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "true"
}

seal "awskms" {
  kms_key_id = "${kms_key}"
}
' > server.hcl

# Wait until consul is up
until consul operator raft list-peers
do
  sleep 1s
done

sleep $[ ( $RANDOM % 10 )  + 1 ]s

nohup vault server -config server.hcl > /var/log/vault.log&
sleep 3
export VAULT_ADDR=http://localhost:8200

# Init the cluster regardless of state, it fails if already set, store 
# root token in json format on primary

vault operator init -format json > /tmp/vault-init
vault status

# if the primary node, do things with the root token
if [ -s /tmp/vault-init ]
then
  export VAULT_TOKEN=$(cat /tmp/vault-init | jq -r '.root_token') 
  vault write /sys/license text=${vault_license}
  vault secrets enable -path="secret" kv
  vault kv put secret/test test=meow
fi