package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
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

	// 事前条件チェック
	checkPrecondition()

	// PID ファイルのチェックと、必要であれば作成
	checkAndCreatePidFile()

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

// PID ファイル処理(開始時)
//
// 1. キャッシュディレクトリ取得、なければ作成
// 2. PID ファイル有無確認
// 3. ファイルに記載されている PID に対応するプロセスの存在確認
//   - 有れば何もせずプログラム終了
//   - 無ければ PID ファイルを削除し、「3.」以降の処理を実行
//
// 4. PID ファイルに PID を記入してファイル作成
// 本当は PID ファイルを消すようにするのがいいのだけど、
// 「3.」でどのみちプロセスの生死確認をしないといけないので手抜きする。
func checkAndCreatePidFile() {

	// 1. キャッシュディレクトリ取得、なければ作成
	pidFile, _, err := getProcessInfoFiles()
	if err != nil {
		panic(err)
	}

	// 2. PID ファイル有無確認
	_, err = os.Stat(pidFile)
	if err == nil {
		// PID ファイルが存在する場合
		// プロセスの有無確認処理を行う
		fmt.Println("pid file found.")
		pidFileContent, err := os.ReadFile(pidFile)
		if err != nil {
			panic(err)
		}

		// 3. ファイルに記載されている PID に対応するプロセスの存在確認

		// 取得した PID を数値に変換
		existedPid, err := strconv.Atoi(string(pidFileContent))
		if err != nil {
			panic(err)
		}

		// 数値に変換した PID を使ってプロセスの存在確認
		fmt.Printf("Test running process PID: %d.\n", existedPid)
		process, err := os.FindProcess(existedPid)
		if err == nil {
			isRunning, err := IsRunnnigProcess(process)
			if err != nil {
				// そもそもチェック処理で失敗
				panic(err)
			}

			if isRunning {
				// プロセス実行中
				fmt.Println("clipboard-receiver already running.")
				os.Exit(0)
			} else {
				// プロセスが実行中でない
				fmt.Println("clipboard-receiver process not found.")
				err = os.Remove(pidFile)
				if err != nil {
					panic(err)
				}
			}
		} else {
			// プロセスが存在していないなら PID ファイルを削除
			fmt.Println("clipboard-receiver process not found.")
			err = os.Remove(pidFile)
			if err != nil {
				panic(err)
			}
		}
	}

	// 4. PID ファイルに PID を記入してファイル作成
	// (既存プロセスが存在する場合、ここまで来る前に Exit する)
	// PID ファイルを作成
	currentPid := os.Getpid()
	fmt.Printf("Start with PID: %d.\n", currentPid)
	err = os.WriteFile(pidFile, []byte(strconv.Itoa(currentPid)), 0600)
	if err != nil {
		panic(err)
	}
}

func getProcessInfoFiles() (string, string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}
	appCacheDir := filepath.Join(userCacheDir, "clipboard-receiver")

	// キャッシュディレクトリを作成
	err = os.MkdirAll(appCacheDir, 744)
	if err != nil {
		return "", "", err
	}

	return filepath.Join(appCacheDir, "pid"),
		filepath.Join(appCacheDir, "port"),
		nil

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
