#/bash/sh
export VERSION=0.26.33
export GOPROXY=direct

sudo apt-get update
sudo apt-get install -y gcc-mingw-w64-i686 gcc-multilib zip
env GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -ldflags "-s -w -extldflags -static -extldflags -static" -buildmode=c-shared -o npc_sdk.dll cmd/npc/sdk.go
env GOOS=linux GOARCH=386 CGO_ENABLED=1 CC=gcc go build -ldflags "-s -w -extldflags -static -extldflags -static" -buildmode=c-shared -o npc_sdk.so cmd/npc/sdk.go
zip -r npc_sdk.zip npc_sdk.dll npc_sdk.so npc_sdk.h

wget https://github.com/upx/upx/releases/download/v3.95/upx-3.95-amd64_linux.tar.xz
tar -xvf upx-3.95-amd64_linux.tar.xz
cp upx-3.95-amd64_linux/upx ./

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static"  ./cmd/npc/npc.go
zip -r linux_amd64_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_386_client.zip npc 


CGO_ENABLED=0 GOOS=freebsd GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r freebsd_386_client.zip npc 


CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r freebsd_amd64_client.zip npc 


CGO_ENABLED=0 GOOS=freebsd GOARCH=arm go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r freebsd_arm_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_arm_v7_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_arm_v6_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_arm_v5_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_arm64_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=mips64 GOMIPS64=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_mips64_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=mips64le GOMIPS64=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_mips64le_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_mipsle_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r linux_mips_client.zip npc 


CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r windows_386_client.zip npc.exe 


CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r windows_amd64_client.zip npc.exe 


CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r darwin_amd64_client.zip npc 


CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/npc/npc.go
zip -r darwin_arm64_client.zip npc 


CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_amd64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_386_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_arm_v5_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_arm_v6_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_arm_v7_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_arm64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=freebsd GOARCH=arm go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r freebsd_arm_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=freebsd GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r freebsd_386_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r freebsd_amd64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_mips_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=mips64 GOMIPS64=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_mips64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=mips64le GOMIPS64=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_mips64le_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r linux_mipsle_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r darwin_amd64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r darwin_arm64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps


CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r windows_amd64_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps.exe


CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go
zip -r windows_386_server.zip conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key  conf/server.pem web/views web/static nps.exe

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update
sudo apt-get -y -o Dpkg::Options::="--force-confnew" install docker-ce
docker --version
docker run --rm -i -w /app -v $(pwd):/app -e ANDROID_HOME=/usr/local/android_sdk -e GOPROXY=direct lucor/fyne-cross:android-latest /app/build.android.sh
git clone https://github.com/cnlh/spksrc.git ~/spksrc
mkdir ~/spksrc/nps && cp -rf ./* ~/spksrc/nps/
docker run -itd --name spksrc --env VERSION=$VERSION -e GOPROXY=direct -v ~/spksrc:/spksrc synocommunity/spksrc /bin/bash
docker exec -it spksrc /bin/bash -c 'cd /spksrc && make setup && cd /spksrc/spk/npc && make arch-x64-7.0'
cp ~/spksrc/packages/npc_x64-7.0_$VERSION-1.spk ./npc_syno.spk


echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
export DOCKER_CLI_EXPERIMENTAL=enabled
docker run --rm --privileged docker/binfmt:66f9012c56a8316f9244ffd7622d7c21c1f6f28d
docker buildx create --use --name mybuilder
docker buildx build --tag ffdfgdfg/nps:$VERSION --tag ffdfgdfg/nps:latest --output type=image,push=true --file Dockerfile.nps --platform=linux/amd64,linux/arm64,linux/386,linux/arm .
docker buildx build --tag ffdfgdfg/npc:$VERSION --tag ffdfgdfg/npc:latest --output type=image,push=true --file Dockerfile.npc --platform=linux/amd64,linux/arm64,linux/386,linux/arm .
