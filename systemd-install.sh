#!/bin/bash

# Script to build and install labns as a systemd service
# Must be run as root
# Requires Go 1.15+

LABNS_ETC_PATH=/etc/labns
LABNS_LOG_PATH=/var/labns
SYSTEMD_PATH=/etc/systemd/system
BIN_PATH=/usr/local/bin/

# build app
go mod download
CGO_ENABLED=0 GOOS=linux go build -o ./bin/main ./cmd/labns/main.go

# install as systemd service
mkdir -p $LABNS_ETC_PATH
cp ./sample-config.json $LABNS_ETC_PATH/labns.json
cp ./systemd/service.conf $LABNS_ETC_PATH/service.conf
cp ./systemd/labns.service $SYSTEMD_PATH/labns.service
cp ./bin/main $BIN_PATH/labns

systemctl daemon-reload