#!/bin/bash

until consul operator raft list-peers; do
  sleep 1s
done

consul license put '${consul_license}' || exit 1

exit 0
