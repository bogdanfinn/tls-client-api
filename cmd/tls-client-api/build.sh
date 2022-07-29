#!/bin/sh

echo 'Build OSX'
GOOS=darwin GOARCH=arm64 go build -o ./../../dist/tls-client-api-darwin-$1 ./main.go

echo 'Build Linux'
GOOS=linux GOARCH=amd64 go build -o ./../../dist/tls-client-api-linux-$1 ./main.go

echo 'Build Windows 32 Bit'
GOOS=windows GOARCH=386 go build -o ./../../dist/tls-client-api-windows-32-$1.exe ./main.go

echo 'Build Windows 64 Bit'
GOOS=windows GOARCH=amd64 go build -o ./../../dist/tls-client-api-windows-64-$1.exe ./main.go