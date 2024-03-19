# clipboard-data-receiver 開発メモ

## 開発環境起動

```sh
docker run -it --rm -v "$(pwd):/work" --workdir /work --name clipboard-data-receiver golang:1.22.1-bookworm
```

## リリースビルド

```sh
apt update
apt install -y libx11-dev
export LD_FLAGS="-s -w"
export ARCH=amd64
GOOS=windows GOARCH=${ARCH} go build -ldflags="${LD_FLAGS}" -trimpath -o build/clipboard-data-receiver.windows-${ARCH}.exe ./main.go
GOOS=linux GOARCH=${ARCH} go build -ldflags="${LD_FLAGS}" -trimpath -o build/clipboard-data-receiver.linux-${ARCH} ./main.go
GOOS=darwin GOARCH=${ARCH} go build -ldflags="${LD_FLAGS}" -trimpath -o build/clipboard-data-receiver.darwin-${ARCH} ./main.go
```

