# clipboard-data-receiver

TCP 経由で受け取ったデータをクリップボードに書き込むプログラム。


# Usage:

```sh
NAME:
   clipboard-data-receiver - Receive clipboard data from remote machine.

USAGE:
   clipboard-data-receiver [global options] 

VERSION:
   1.0.0

GLOBAL OPTIONS:
   --port value, -p value  listen port. (default: 8733)
   --address value         listen address. (default: "0.0.0.0")
   --license               show licensesa.
   --help, -h              show help
   --version, -v           print the version
```

# Example:

以下例のように、 `clipboard-data-receiver` が待ち受けているポートにデータを送信し、
コネクションを閉じると、そのコネクション内で受信したデータをクリップボードへ反映する。

Start clipboard-data-receiver:

```sh
./clipboard-data-receiver --port 8733
```

Send clipboard data:

```sh
echo "YANK_TEXT" | nc -q 0 localhost 8733
```


# Install:

[binary download from Release](https://github.com/mikoto2000/clipboard-data-receiver/releases) or `go install` command.

```sh
go install github.com/mikoto2000/clipboard-data-receiver@latest
```


# License:

Copyright (C) 2024 mikoto2000

This software is released under the MIT License, see LICENSE

このソフトウェアは MIT ライセンスの下で公開されています。 LICENSE を参照してください。


# Author:

mikoto2000 <mikoto2000@gmail.com>


