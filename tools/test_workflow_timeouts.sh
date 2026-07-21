#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-or-later
set -eu
unset CDPATH

root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
fixtures=$(mktemp -d)
trap 'rm -rf "$fixtures"' EXIT INT TERM

printf '%s\n' 'jobs:' '  test:' '    timeout-minutes: 15' '    runs-on: ubuntu-latest' '    steps: []' >"$fixtures/valid.yml"
"$root/tools/check_workflow_timeouts.sh" "$fixtures/valid.yml" >/dev/null

printf '%s\n' 'jobs:' '  test:' '    runs-on: ubuntu-latest' '    steps: []' >"$fixtures/missing.yml"
if "$root/tools/check_workflow_timeouts.sh" "$fixtures/missing.yml" >/dev/null 2>&1; then
	echo "workflow timeout checker accepted a job without a timeout" >&2
	exit 1
fi

printf '%s\n' 'jobs:' '  test:' '    timeout-minutes: 16' '    runs-on: ubuntu-latest' '    steps: []' >"$fixtures/too-long.yml"
if "$root/tools/check_workflow_timeouts.sh" "$fixtures/too-long.yml" >/dev/null 2>&1; then
	echo "workflow timeout checker accepted a timeout greater than 15 minutes" >&2
	exit 1
fi

dollar='$'
printf '%s\n' 'jobs:' '  test:' "    timeout-minutes: ${dollar}{{ matrix.timeout }}" '    runs-on: ubuntu-latest' '    steps: []' >"$fixtures/expression.yml"
if "$root/tools/check_workflow_timeouts.sh" "$fixtures/expression.yml" >/dev/null 2>&1; then
	echo "workflow timeout checker accepted a non-literal timeout" >&2
	exit 1
fi

echo "sharecrop: GitHub Actions job timeout checker tests passed"
