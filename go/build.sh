#!/usr/bin/env bash
set -euo pipefail

APP_NAME="imgtool"
VERSION="${1:-1.0.0}"
OUT_DIR="build"

echo "==> 应用名: ${APP_NAME}"
echo "==> 版本号: ${VERSION}"
echo "==> 输出目录: ${OUT_DIR}"

mkdir -p "${OUT_DIR}"

LDFLAGS="-s -w -X main.Version=${VERSION}"

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

for platform in "${platforms[@]}"; do
  IFS="/" read -r GOOS GOARCH <<< "${platform}"

  output="${OUT_DIR}/${APP_NAME}_${GOOS}_${GOARCH}"
  if [[ "${GOOS}" == "windows" ]]; then
    output="${output}.exe"
  fi

  echo "==> 编译 ${GOOS}/${GOARCH} -> ${output}"
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" \
    go build -trimpath -ldflags="${LDFLAGS}" -o "${output}" main.go
done

echo "==> 全部编译完成"
echo "==> 输出目录: ${OUT_DIR}"