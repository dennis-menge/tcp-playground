#!/bin/bash
apt update 
DEBIAN_FRONTEND=noninteractive TZ=Etc/UTC apt install git golang-go -y
mkdir goroot
export GOPATH=$(pwd)/goroot
git clone https://github.com/dennis-menge/tcp-playground.git
cd tcp-playground  || exit 1
GOCACHE=$(pwd) go build tcpserver.go
cp tcpserver /usr/local/bin
cp tcpserver.service /etc/systemd/system/
systemctl daemon-reload
systemctl start tcpserver.service