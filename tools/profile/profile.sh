#!/usr/bin/env bash
# Copyright IBM Corp. 2021, 2025
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

go tool pprof -proto cpu.pprof.scenario.list cpu.pprof.sample.observe cpu.pprof.validate > default.pprof
if test -f default.pgo; then
  go tool pprof -proto default.pgo default.pprof > combined.pgo
  mv combined.pgo default.pgo
else
  mv default.pprof default.pgo
fi
