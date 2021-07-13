#!/usr/bin/env bash

# The Vault smoke test. This is a modified version of the license checking script
# https://github.com/hashicorp/vault-enterprise/blob/master/scripts/testing/test-vault-license.sh

set -e

binpath=${vault_install_dir}/vault
edition=${vault_edition}
version=${vault_version}
release="$version+$edition"

# OSS editions don't require or support licenses.
if test "$edition" == "oss"; then
  exit 0
fi

# Vault >= 1.8 no longer includes built-in license so we can move on.
if [[ "$release" = *1.[8-9].*+ent* ]]; then
  exit 0
fi

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

echo "Installing required package(s) for the smoke test"
# Make sure cloud-init is not modifying our sources list while we're trying
# to install jq.
retry 5 grep ec2 /etc/apt/sources.list

cd /tmp
retry 5 sudo apt update
retry 5 sudo apt install -y jq

fail() {
	echo "$1" 1>&2
	exit 1
}

license_expected="temporary"
case "$release" in
	*+ent) ;;
	*+ent.hsm) ;;
	*+pro) license_expected="permanent";;
	*+prem.hsm) license_expected="permanent";;
	*+prem) license_expected="permanent";;
  *) fail "($release) file doesn't match any known license types"
esac

features_expected=""
case "$release" in
	*1.2.*+ent*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP"]';;
	*1.3.*+ent*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation"]';;
	*1.4.*+ent*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation", "Transform Secrets Engine"]';;
	*1.5.*+ent*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation", "Transform Secrets Engine", "Lease Count Quotas"]';;
	*1.[6-8].*+ent*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation", "Transform Secrets Engine", "Lease Count Quotas", "Key Management Secrets Engine", "Automated Snapshots"]';;

	*1.[2-4].*+pro*) features_expected='["DR Replication", "Performance Standby", "Namespaces"]';;
	*1.5.*+pro*) features_expected='["DR Replication", "Performance Standby", "Namespaces", "Lease Count Quotas"]';;
	*1.[6-8].*+pro*) features_expected='["DR Replication", "Performance Standby", "Namespaces", "Lease Count Quotas", "Automated Snapshots"]';;

	*1.2.*+prem*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP"]';;
	*1.[3-4].*+prem*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation"]';;
	*1.5.*+prem*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation", "Lease Count Quotas"]';;
	*1.[6-8].*+prem*) features_expected='["HSM", "Performance Replication", "DR Replication", "MFA", "Sentinel", "Seal Wrapping", "Control Groups", "Performance Standby", "Namespaces", "KMIP", "Entropy Augmentation", "Lease Count Quotas", "Automated Snapshots"]';;

	*) fail "zip file doesn't match any known feature types"
esac

test -x "$binpath" || fail "unable to locate vault binary at $binpath"

export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='${vault_token}'
retry 5 "$binpath" status > /dev/null 2>&1

license_id=$("$binpath" read -format=json sys/license | jq -Mr .data.license_id)
test "$license_id" = "$license_expected" || fail "expected license_id=$license_expected, got: $license_id"

test true == $("$binpath" read -format=json sys/license | jq -Mr --argjson expected "$features_expected" '.data.features == $expected') ||
	fail "expected features=$features_expected, got: $("$binpath" read -format=json sys/license | jq -Mr '.data.features')"
