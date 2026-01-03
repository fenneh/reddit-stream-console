#!/bin/bash

echo "Building Reddit Stream Console..."

# Check if go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go 1.22+ and try again."
    exit 1
fi

mkdir -p bin

echo "Compiling binary..."
go build -o bin/reddit-stream-console ./cmd/reddit-stream-console

echo "Build complete! Run './bin/reddit-stream-console' to start."
