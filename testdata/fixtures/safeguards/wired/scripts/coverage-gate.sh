#!/bin/sh
set -eu
go tool cover -func="$1" | tail -n 1
