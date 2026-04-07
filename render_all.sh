#!/usr/bin/env bash
set -euox pipefail

renderers=(
  "plasma"
  "mandelbrot"
  "tunnel"
)

mkdir -p videos

for renderer in "${renderers[@]}"; do
  go run . -renderer "$renderer" -out "videos/${renderer}.mp4" "${@}"
done
