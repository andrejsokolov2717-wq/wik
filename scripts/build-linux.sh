#!/usr/bin/env bash
# Сборка Sweepy для Linux.
# Зависимости: golang (>=1.22), gcc, libgl1-mesa-dev, xorg-dev.
set -euo pipefail

VERSION="${VERSION:-1.0.0}"
mkdir -p dist
CGO_ENABLED=1 go build -ldflags "-s -w -X sweepy/internal/build.Version=${VERSION}" \
  -o "dist/sweepy-linux-amd64" .

echo "Готово: dist/sweepy-linux-amd64"
