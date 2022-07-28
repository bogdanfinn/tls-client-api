#!/bin/sh

GOOS=darwin GOARCH=arm64 go build -o ./../../dist/tls-client-api-darwin ./main.go

GOOS=linux GOARCH=amd64 go build -o ./../../dist/tls-client-api-linux ./main.go

GOOS=windows GOARCH=386 go build -o ./../../dist/tls-client-api-windows-32.exe ./main.go

GOOS=windows GOARCH=amd64 go build -o ./../../dist/tls-client-api-windows-64.exe ./main.go