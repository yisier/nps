
#!/bin/bash

# 构建nps/npc客户端下载文件的脚本
# 该脚本会生成各种平台的客户端文件，用于Web界面的下载页面

set -e  # 遇到错误时退出

echo "开始构建客户端下载文件..."

# 创建下载目录
DOWNLOADS_DIR="downloads"
WEB_DOWNLOADS_DIR="web/static/downloads"

mkdir -p $DOWNLOADS_DIR
mkdir -p $WEB_DOWNLOADS_DIR

# 定义版本号
VERSION=${VERSION:-"0.26.25"}

echo "构建Linux客户端..."
# Linux客户端
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags -static" -o $DOWNLOADS_DIR/linux_amd64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/linux_amd64/
tar -czvf $WEB_DOWNLOADS_DIR/linux_amd64_client.tar.gz -C $DOWNLOADS_DIR/linux_amd64 .

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -extldflags -static" -o $DOWNLOADS_DIR/linux_arm64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/linux_arm64/
tar -czvf $WEB_DOWNLOADS_DIR/linux_arm64_client.tar.gz -C $DOWNLOADS_DIR/linux_arm64 .

echo "构建Windows客户端..."
# Windows客户端
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -extldflags -static" -o $DOWNLOADS_DIR/windows_amd64/npc.exe ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/windows_amd64/
zip -j $WEB_DOWNLOADS_DIR/windows_amd64_client.zip $DOWNLOADS_DIR/windows_amd64/npc.exe $DOWNLOADS_DIR/windows_amd64/npc.conf $DOWNLOADS_DIR/windows_amd64/multi_account.conf

CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-s -w -extldflags -static" -o $DOWNLOADS_DIR/windows_386/npc.exe ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/windows_386/
zip -j $WEB_DOWNLOADS_DIR/windows_386_client.zip $DOWNLOADS_DIR/windows_386/npc.exe $DOWNLOADS_DIR/windows_386/npc.conf $DOWNLOADS_DIR/windows_386/multi_account.conf

echo "构建macOS客户端..."
# macOS客户端
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags -static" -o $DOWNLOADS_DIR/darwin_amd64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/darwin_amd64/
tar -czvf $WEB_DOWNLOADS_DIR/darwin_amd64_client.tar.gz -C $DOWNLOADS_DIR/darwin_amd64 .

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags -static" -o $DOWNLOADS_DIR/darwin_arm64/npc ./cmd/npc/npc.go
cp conf/npc.conf conf/multi_account.conf $DOWNLOADS_DIR/darwin_arm64/
tar -czvf $WEB_DOWNLOADS_DIR/darwin_arm64_client.tar.gz -C $DOWNLOADS_DIR/darwin_arm64 .


echo "构建完成！客户端文件已生成到相应目录。"
echo "Web下载目录: $WEB_DOWNLOADS_DIR"
echo "原始文件目录: $DOWNLOADS_DIR"