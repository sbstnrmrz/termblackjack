#!/bin/bash

platforms=("windows/amd64" "linux/amd64" "darwin/arm64")

for platform in ${platforms[@]}
do
    IFS='/'
    read -ra tokens <<< "${platform}"

    OS=${tokens[0]}
    ARCH=${tokens[1]}
    dir="bin/"
    if [ $OS = "darwin" ]; then
        dir+="mac"
    else
        dir+="$OS"
    fi

    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
    fi

    echo "binary for platform: $OS/$ARCH created in $dir" 
    GOOS=$OS GOARCH=$ARCH go build -o "$dir" .
done
