#!/usr/bin/env bash

set -e

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

function fail {
	echo "$1" 1>&2
	exit 1
}

binpath=${vault_install_dir}/vault
instanceid=${instance_id}

# Only enable the secrets engine on the first node
if test $instanceid != '0'; then
  exit 0
fi

test -x "$binpath" || fail "unable to locate vault binary at $binpath"

export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='${vault_token}'

retry 5 "$binpath" status > /dev/null 2>&1
retry 5 $binpath secrets enable -path="secret" kv
