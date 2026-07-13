export GOPROXY=direct

sudo apt-get update
sudo apt-get install -y gcc-mingw-w64-i686 gcc-multilib zip
env GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -ldflags "-s -w -extldflags -static -extldflags -static" -buildmode=c-shared -o npc_sdk.dll cmd/npc/sdk.go
env GOOS=linux GOARCH=386 CGO_ENABLED=1 CC=gcc go build -ldflags "-s -w -extldflags -static -extldflags -static" -buildmode=c-shared -o npc_sdk.so cmd/npc/sdk.go
zip -r npc_sdk.zip npc_sdk.dll npc_sdk.so npc_sdk.h

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

zip -r linux_amd64_server.zip nps

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_386_server.zip nps

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_arm_v5_server.zip nps

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_arm_v6_server.zip nps

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_arm_v7_server.zip nps


CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_arm64_server.zip nps


CGO_ENABLED=0 GOOS=freebsd GOARCH=arm go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r freebsd_arm_server.zip nps


CGO_ENABLED=0 GOOS=freebsd GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r freebsd_386_server.zip nps


CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r freebsd_amd64_server.zip nps



CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_mips_server.zip nps


CGO_ENABLED=0 GOOS=linux GOARCH=mips64 GOMIPS64=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_mips64_server.zip nps


CGO_ENABLED=0 GOOS=linux GOARCH=mips64le GOMIPS64=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_mips64le_server.zip nps


CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r linux_mipsle_server.zip nps



CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r darwin_amd64_server.zip nps


CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r darwin_arm64_server.zip nps



CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r windows_amd64_server.zip nps.exe


CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-s -w -extldflags -static -extldflags -static" ./cmd/nps/nps.go

zip -r windows_386_server.zip nps.exe