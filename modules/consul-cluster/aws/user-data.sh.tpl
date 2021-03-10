#!/bin/bash
cd /tmp
apt update
apt install -y unzip awscli jq
wget ${package_url}
unzip consul_*_linux_amd64.zip -d /usr/local/bin

nohup consul agent -retry-join "provider=aws tag_key=Type tag_value=consul-server" -data-dir=/tmp/consul -server -bootstrap-expect=3 -log-file=/var/log/consul.log -ui -client 0.0.0.0&

until consul operator raft list-peers
do
  sleep 1s
done
consul license put "${consul_license}"
echo consul license put "${consul_license}"