#!/usr/bin/env bash
set -euo pipefail

go work sync

go build -o bin/burrowctl ./burrowctl
go build -o bin/burrowd ./burrowd

echo "Built bin/burrowctl and bin/burrowd"
