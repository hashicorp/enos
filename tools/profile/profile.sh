#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


# NOTE: The paths in this script assume it's being executed from the root of the repository.

set -ex

[ ! -f dist/enos ] && echo "you must build the enos binary to create profiles" 2>&1 && exit 1

dist/enos scenario list  --format json --chdir ./acceptance/scenarios/build_pgo/ --profile | jq '.scenarios | length'
mv cpu.pprof cpu.pprof.scenario.list

dist/enos scenario sample observe complex --max 10 --chdir ./acceptance/scenarios/build_pgo/ --profile
mv cpu.pprof cpu.pprof.sample.observe

dist/enos scenario validate --chdir ./acceptance/scenarios/build_pgo/ --profile
mv cpu.pprof cpu.pprof.validate

go tool pprof -proto cpu.pprof.scenario.list cpu.pprof.sample.observe cpu.pprof.validate > default.pgo
if test -f default.pgo; then
  cp default.pgo default.pprof
  go tool pprof -proto cpu.pprof.scenario.list cpu.pprof.sample.observe cpu.pprof.validate default.pprof > default.pgo
  exit 0
fi
