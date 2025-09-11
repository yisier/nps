#!/bin/bash

# 构建nps/npc客户端和服务端下载文件的脚本
# 该脚本会生成各种平台的客户端和服务端文件，用于Web界面的下载页面

# 版本号控制（从VERSION文件读取）
if [ -f "VERSION" ]; then
    VERSION=$(cat VERSION)
else
    VERSION="0.26.25"
fi

set -e  # 遇到错误时退出

echo "开始构建客户端和服务端下载文件，版本: $VERSION"

# 创建下载目录
DOWNLOADS_DIR="downloads"
WEB_DOWNLOADS_DIR="web/static/downloads"
DIST_DIR="dist"

mkdir -p $DOWNLOADS_DIR
mkdir -p $WEB_DOWNLOADS_DIR
mkdir -p $DIST_DIR

echo "构建Linux客户端..."
# Linux客户端
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/linux_amd64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/linux_amd64/
tar -czvf $WEB_DOWNLOADS_DIR/linux_amd64_client.tar.gz -C $DOWNLOADS_DIR/linux_amd64 .
cp $WEB_DOWNLOADS_DIR/linux_amd64_client.tar.gz $DIST_DIR/

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/linux_arm64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/linux_arm64/
tar -czvf $WEB_DOWNLOADS_DIR/linux_arm64_client.tar.gz -C $DOWNLOADS_DIR/linux_arm64 .
cp $WEB_DOWNLOADS_DIR/linux_arm64_client.tar.gz $DIST_DIR/

echo "构建Windows客户端..."
# Windows客户端
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/windows_amd64/npc.exe ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/windows_amd64/
zip -j $WEB_DOWNLOADS_DIR/windows_amd64_client.zip $DOWNLOADS_DIR/windows_amd64/npc.exe $DOWNLOADS_DIR/windows_amd64/npc.conf $DOWNLOADS_DIR/windows_amd64/multi_account.conf
cp $WEB_DOWNLOADS_DIR/windows_amd64_client.zip $DIST_DIR/

CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/windows_386/npc.exe ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/windows_386/
zip -j $WEB_DOWNLOADS_DIR/windows_386_client.zip $DOWNLOADS_DIR/windows_386/npc.exe $DOWNLOADS_DIR/windows_386/npc.conf $DOWNLOADS_DIR/windows_386/multi_account.conf
cp $WEB_DOWNLOADS_DIR/windows_386_client.zip $DIST_DIR/

echo "构建macOS客户端..."
# macOS客户端
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/darwin_amd64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/darwin_amd64/
tar -czvf $WEB_DOWNLOADS_DIR/darwin_amd64_client.tar.gz -C $DOWNLOADS_DIR/darwin_amd64 .
cp $WEB_DOWNLOADS_DIR/darwin_amd64_client.tar.gz $DIST_DIR/

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/darwin_arm64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/darwin_arm64/
tar -czvf $WEB_DOWNLOADS_DIR/darwin_arm64_client.tar.gz -C $DOWNLOADS_DIR/darwin_arm64 .
cp $WEB_DOWNLOADS_DIR/darwin_arm64_client.tar.gz $DIST_DIR/

echo "构建Linux服务端..."
# Linux服务端
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/linux_amd64_server/nps ./cmd/nps/nps.go
cp conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key conf/server.pem $DOWNLOADS_DIR/linux_amd64_server/
cp -r web/views $DOWNLOADS_DIR/linux_amd64_server/
# 只复制必要的静态文件，避免循环包含
mkdir -p $DOWNLOADS_DIR/linux_amd64_server/web/static
cp -r web/static/css web/static/js web/static/img web/static/webfonts web/static/page $DOWNLOADS_DIR/linux_amd64_server/web/static/
tar -czvf $WEB_DOWNLOADS_DIR/linux_amd64_server.tar.gz -C $DOWNLOADS_DIR/linux_amd64_server .
cp $WEB_DOWNLOADS_DIR/linux_amd64_server.tar.gz $DIST_DIR/

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/linux_arm64_server/nps ./cmd/nps/nps.go
cp conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key conf/server.pem $DOWNLOADS_DIR/linux_arm64_server/
cp -r web/views $DOWNLOADS_DIR/linux_arm64_server/
# 只复制必要的静态文件，避免循环包含
mkdir -p $DOWNLOADS_DIR/linux_arm64_server/web/static
cp -r web/static/css web/static/js web/static/img web/static/webfonts web/static/page $DOWNLOADS_DIR/linux_arm64_server/web/static/
tar -czvf $WEB_DOWNLOADS_DIR/linux_arm64_server.tar.gz -C $DOWNLOADS_DIR/linux_arm64_server .
cp $WEB_DOWNLOADS_DIR/linux_arm64_server.tar.gz $DIST_DIR/

echo "构建Windows服务端..."
# Windows服务端
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/windows_amd64_server/nps.exe ./cmd/nps/nps.go
cp conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key conf/server.pem $DOWNLOADS_DIR/windows_amd64_server/
cp -r web/views $DOWNLOADS_DIR/windows_amd64_server/
# 只复制必要的静态文件，避免循环包含
mkdir -p $DOWNLOADS_DIR/windows_amd64_server/web/static
cp -r web/static/css web/static/js web/static/img web/static/webfonts web/static/page $DOWNLOADS_DIR/windows_amd64_server/web/static/
zip -r $WEB_DOWNLOADS_DIR/windows_amd64_server.zip $DOWNLOADS_DIR/windows_amd64_server
cp $WEB_DOWNLOADS_DIR/windows_amd64_server.zip $DIST_DIR/

CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/windows_386_server/nps.exe ./cmd/nps/nps.go
cp conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key conf/server.pem $DOWNLOADS_DIR/windows_386_server/
cp -r web/views $DOWNLOADS_DIR/windows_386_server/
# 只复制必要的静态文件，避免循环包含
mkdir -p $DOWNLOADS_DIR/windows_386_server/web/static
cp -r web/static/css web/static/js web/static/img web/static/webfonts web/static/page $DOWNLOADS_DIR/windows_386_server/web/static/
zip -r $WEB_DOWNLOADS_DIR/windows_386_server.zip $DOWNLOADS_DIR/windows_386_server
cp $WEB_DOWNLOADS_DIR/windows_386_server.zip $DIST_DIR/

echo "构建macOS服务端..."
# macOS服务端
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/darwin_amd64_server/nps ./cmd/nps/nps.go
cp conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key conf/server.pem $DOWNLOADS_DIR/darwin_amd64_server/
cp -r web/views $DOWNLOADS_DIR/darwin_amd64_server/
# 只复制必要的静态文件，避免循环包含
mkdir -p $DOWNLOADS_DIR/darwin_amd64_server/web/static
cp -r web/static/css web/static/js web/static/img web/static/webfonts web/static/page $DOWNLOADS_DIR/darwin_amd64_server/web/static/
tar -czvf $WEB_DOWNLOADS_DIR/darwin_amd64_server.tar.gz -C $DOWNLOADS_DIR/darwin_amd64_server .
cp $WEB_DOWNLOADS_DIR/darwin_amd64_server.tar.gz $DIST_DIR/

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags -static -X ehang.io/nps/lib/version.VERSION=$VERSION" -o $DOWNLOADS_DIR/darwin_arm64_server/nps ./cmd/nps/nps.go
cp conf/nps.conf conf/tasks.json conf/clients.json conf/hosts.json conf/server.key conf/server.pem $DOWNLOADS_DIR/darwin_arm64_server/
cp -r web/views $DOWNLOADS_DIR/darwin_arm64_server/
# 只复制必要的静态文件，避免循环包含
mkdir -p $DOWNLOADS_DIR/darwin_arm64_server/web/static
cp -r web/static/css web/static/js web/static/img web/static/webfonts web/static/page $DOWNLOADS_DIR/darwin_arm64_server/web/static/
tar -czvf $WEB_DOWNLOADS_DIR/darwin_arm64_server.tar.gz -C $DOWNLOADS_DIR/darwin_arm64_server .
cp $WEB_DOWNLOADS_DIR/darwin_arm64_server.tar.gz $DIST_DIR/

echo "构建完成！客户端和服务端文件已生成到相应目录。"
echo "Web下载目录: $WEB_DOWNLOADS_DIR"
echo "分发目录: $DIST_DIR"
echo "原始文件目录: $DOWNLOADS_DIR"