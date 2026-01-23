#!/bin/bash
set -euo pipefail

cd /go
apt-get update
apt-get install -y libegl1-mesa-dev libgles2-mesa-dev libx11-dev xorg-dev
go install fyne.io/fyne/v2/cmd/fyne@latest
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
rm cmd/npc/sdk.go
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
test -f npc.apk
mv npc.apk /app/android_client.apk
echo "android build success"
