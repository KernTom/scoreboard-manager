#!/bin/bash
echo "🔨 Building Windows EXE..."
GOOS=windows GOARCH=amd64 go build -o scoreboard-admin.exe ./cmd/admin
echo "🚀 Starting via Wine..."
wine scoreboard-admin.exe