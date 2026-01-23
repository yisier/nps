#!/bin/bash
set -euo pipefail

cd /go
if grep -q "buster" /etc/apt/sources.list; then
  sed -i 's|http://deb.debian.org/debian|http://archive.debian.org/debian|g' /etc/apt/sources.list
  sed -i 's|http://security.debian.org/debian-security|http://archive.debian.org/debian-security|g' /etc/apt/sources.list
  echo 'Acquire::Check-Valid-Until "false";' > /etc/apt/apt.conf.d/99no-check-valid
fi
apt-get update
apt-get install -y libegl1-mesa-dev libgles2-mesa-dev libx11-dev xorg-dev ca-certificates curl

GO_VERSION="${GO_VERSION:-1.22.7}"
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tgz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tgz
export PATH="/usr/local/go/bin:$PATH"
export GOTOOLCHAIN=auto
go version

GO111MODULE=on go install fyne.io/fyne/v2/cmd/fyne@latest
export PATH="$PATH:$(go env GOPATH)/bin"
#mkdir -p /go/src/fyne.io
#cd src/fyne.io
#git clone https://github.com/fyne-io/fyne.git
#cd fyne
#git checkout v1.2.0
#go install -v ./cmd/fyne
#fyne package -os android fyne.io/fyne/cmd/hello
echo "fyne install success"
mkdir -p /go/src/ehang.io/nps
cp -R /app/* /go/src/ehang.io/nps
cd /go/src/ehang.io/nps
#go get -u fyne.io/fyne fyne.io/fyne/cmd/fyne
rm -f cmd/npc/sdk.go
#go get -u ./...
#go mod tidy
#rm -rf /go/src/golang.org/x/mobile
echo "tidy success"
cd /go/src/ehang.io/nps
go mod vendor
cd vendor
cp -R * /go/src
cd ..
rm -rf vendor
#rm -rf ~/.cache/*
echo "vendor success"
cd gui/npc
fyne package -appID org.nps.client -os android -icon ../../docs/logo.png
apk="$(ls -1 *.apk 2>/dev/null | head -n 1)"
if [ -z "$apk" ]; then
  echo "No apk produced"
  ls -la
  exit 1
fi
mv "$apk" /app/android_client.apk
echo "android build success"
