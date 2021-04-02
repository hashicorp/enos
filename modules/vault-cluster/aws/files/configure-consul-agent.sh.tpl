#!/bin/bash

set -eux

export fulllog="/tmp/vault_consul_agent.log"

exec > $fulllog 2>&1

# Configure dnsmasq to use consul
if [ ! -e  /etc/dnsmasq.d/consul.conf ]
then
    echo "server=/consul/127.0.0.1#8600
    rev-server=10.0.0.0/8,127.0.0.1#8600" |sudo tee /etc/dnsmasq.d/consul.conf
    sudo service dnsmasq restart
fi

#IPs of consul nodes, we could also use AWS lookup
IPS="${consul_ips}"

# Make sure consul agent is started and joins
sudo nohup consul agent -retry-join $${IPS// / -join } -data-dir /tmp > consul.out&


# Wait until consul is up
until consul operator raft list-peers
do
  sleep 1s
done

consul operator raft list-peers &>> /tmp/consul-agent.out