package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/urfave/cli/v2"
	"golang.design/x/clipboard"
)

const DEFAULT_ADDRESS = "0.0.0.0"
const DEFAULT_PORT = 8733
const RECEIVE_BUFFER_SIZE = 1024

const FLAG_NAME_ADDRESS = "address"
const FLAG_NAME_PORT = "port"
const FLAG_NAME_LICENSE = "license"

//go:embed LICENSE
var license string

//go:embed NOTICE
var notice string

func main() {

	checkPrecondition()

	app := (&cli.App{
		Name:                   "clipboard-data-receiver",
		Usage:                  "Receive clipboard data from remote machine.",
		Version:                "1.0.0",
		UseShortOptionHandling: true,
		HideHelpCommand:        true,
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    FLAG_NAME_PORT,
				Aliases: []string{"p"},
				Value:   DEFAULT_PORT,
				Usage:   "listen port.",
			},
			&cli.StringFlag{
				Name:  FLAG_NAME_ADDRESS,
				Value: DEFAULT_ADDRESS,
				Usage: "listen address.",
			},
			&cli.BoolFlag{
				Name:               FLAG_NAME_LICENSE,
				Value:              false,
				DisableDefaultText: true,
				Usage:              "show licensesa.",
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.Bool(FLAG_NAME_LICENSE) {
				fmt.Println(license)
				fmt.Println()
				fmt.Println(notice)
				return nil
			}

			startListen(cCtx.String(FLAG_NAME_ADDRESS), strconv.Itoa(cCtx.Int(FLAG_NAME_PORT)))
			return nil
		},
	})

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

func checkPrecondition() {
	// clipboard が利用可能か確認
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
}

func startListen(address, port string) {
	// Listen 開始
	listener, err := net.Listen("tcp", address+":"+port)
	if err != nil {
		panic(err)
	}

	// 接続を待ち受け、クライアントからの接続があったら
	// 接続時処理(`handleConnection`)を開始
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		// 接続時処理を開始
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {

	// コネクションクローズ時処理を defer で定義
	defer func() {
		conn.Close()
	}()

	var receivedData bytes.Buffer

	// データ受信
	buf := make([]byte, RECEIVE_BUFFER_SIZE)
	for {
		readSize, err := conn.Read(buf)
		if readSize == 0 {
			break
		}
		if err != nil {
			panic(err)
		}

		receivedData.Write(buf[0:readSize])
	}

	writeToClipboard(receivedData.Bytes())
}

func writeToClipboard(data []byte) {
	// クリップボードへ貼り付け
	clipboard.Write(clipboard.FmtText, data)
}
