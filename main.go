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

const APP_NAME = "clipboard-data-receiver"

const DEFAULT_ADDRESS = "0.0.0.0"
const DEFAULT_PORT = 8733
const RECEIVE_BUFFER_SIZE = 1024

const FLAG_NAME_ADDRESS = "address"
const FLAG_NAME_PORT = "port"
const FLAG_NAME_LICENSE = "license"
const FLAG_NAME_RANDOM_PORT = "random-port"
const FLAG_NAME_PID_FILE = "pid-file"
const FLAG_NAME_PORT_FILE = "port-file"

const OUTPUT_TEMPLATE = "{\n  \"pid\": %d,\n  \"address\": \"%s\",\n  \"port\": %d\n}\n"

//go:embed LICENSE
var license string

//go:embed NOTICE
var notice string

func main() {

	// 事前条件チェック
	checkPrecondition()

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	appCacheDir := filepath.Join(userCacheDir, APP_NAME)
	defaultPidFilePath := filepath.Join(appCacheDir, "pid")
	defaultPortFilePath := filepath.Join(appCacheDir, "port")

	app := (&cli.App{
		Name:                   "clipboard-data-receiver",
		Usage:                  "Receive clipboard data from remote machine.",
		Version:                "3.0.0",
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
			&cli.BoolFlag{
				Name:  FLAG_NAME_RANDOM_PORT,
				Value: false,
				Usage: "use a random available port.",
			},
			&cli.StringFlag{
				Name:  FLAG_NAME_PID_FILE,
				Value: defaultPidFilePath,
				Usage: "pid file path.",
			},
			&cli.StringFlag{
				Name:  FLAG_NAME_PORT_FILE,
				Value: defaultPortFilePath,
				Usage: "port file path.",
			},
		},
		Action: func(cCtx *cli.Context) error {

			if cCtx.Bool(FLAG_NAME_LICENSE) {
				fmt.Println(license)
				fmt.Println()
				fmt.Println(notice)
				return nil
			}

			pidFile := cCtx.String(FLAG_NAME_PID_FILE)
			portFile := cCtx.String(FLAG_NAME_PORT_FILE)

			pidFileDir := filepath.Dir(pidFile)
			portFileDir := filepath.Dir(portFile)

			// 各ファイルを格納するディレクトリを作成
			err = os.MkdirAll(pidFileDir, 0744)
			if err != nil {
				panic(err)
			}
			err = os.MkdirAll(portFileDir, 0744)
			if err != nil {
				panic(err)
			}

			alreadyRunning, pid, err := checkAndCreatePidFile(pidFile)
			if err != nil {
				panic(err)
			}

			address := cCtx.String(FLAG_NAME_ADDRESS)
			port := cCtx.Int(FLAG_NAME_PORT)

			if alreadyRunning {
				// port ファイルからポート番号を取得
				port, err = getPort(portFile)
				if err != nil {
					panic(err)
				}
				fmt.Printf(OUTPUT_TEMPLATE, pid, address, port)
				os.Exit(0)
			}

			if cCtx.Bool(FLAG_NAME_RANDOM_PORT) {
				port = getRandomPort()
				savePortToCache(portFile, port)
			}

			fmt.Printf(OUTPUT_TEMPLATE, pid, address, port)

			startListen(address, strconv.Itoa(port))
			return nil
		},
	})

	err = app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

// ポートファイルからポート番号を取得する
func getPort(portFilePath string) (int, error) {

	// port ファイルからポート番号を取得
	portFileContent, err := os.ReadFile(portFilePath)
	if err != nil {
		return 0, err
	}

	// 取得した内容を int に変換
	port, err := strconv.Atoi(string(portFileContent))
	if err != nil {
		return 0, err
	}

	return port, nil
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
// 「3.」でどのみちプロセスの生死確認をしないといけないので、
// それに任せるようにする
//
// alreadyRunning, pid, error を返却する。
func checkAndCreatePidFile(pidFile string) (bool, int, error) {
	_, err := os.Stat(pidFile)
	if err == nil {
		// PID ファイルが存在する場合
		// プロセスの有無確認処理を行う
		fmt.Fprintln(os.Stderr, "pid file found.")
		pidFileContent, err := os.ReadFile(pidFile)
		if err != nil {
			return false, 0, err
		}

		// 3. ファイルに記載されている PID に対応するプロセスの存在確認

		// 取得した PID を数値に変換
		existedPid, err := strconv.Atoi(string(pidFileContent))
		if err != nil {
			return false, existedPid, err
		}

		// 数値に変換した PID を使ってプロセスの存在確認
		fmt.Fprintf(os.Stderr, "Test running process PID: %d.\n", existedPid)
		process, err := os.FindProcess(existedPid)
		if err == nil {
			isRunning, err := IsRunningProcess(process)
			if err != nil {
				// そもそもチェック処理で失敗
				return false, process.Pid, err
			}

			if isRunning {
				// プロセス実行中
				fmt.Fprintln(os.Stderr, "clipboard-receiver already running.")
				return true, process.Pid, nil
			} else {
				// プロセスが実行中でない
				fmt.Fprintln(os.Stderr, "clipboard-receiver process not found.")
				err = os.Remove(pidFile)
				if err != nil {
					return false, process.Pid, err
				}
			}
		} else {
			// プロセスが存在していないなら PID ファイルを削除
			fmt.Fprintln(os.Stderr, "clipboard-receiver process not found.")
			err = os.Remove(pidFile)
			if err != nil {
				return false, 0, err
			}
		}
	}

	// 4. PID ファイルに PID を記入してファイル作成
	// (既存プロセスが存在する場合、ここまで来る前に Exit する)
	// PID ファイルを作成
	currentPid := os.Getpid()
	fmt.Fprintf(os.Stderr, "Start with PID: %d.\n", currentPid)
	err = os.WriteFile(pidFile, []byte(strconv.Itoa(currentPid)), 0600)
	if err != nil {
		panic(err)
	}

	return false, currentPid, nil
}

func getProcessInfoFiles() (string, string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}
	appCacheDir := filepath.Join(userCacheDir, "clipboard-receiver")

	// キャッシュディレクトリを作成
	err = os.MkdirAll(appCacheDir, 0744)
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
	fmt.Fprintf(os.Stderr, "Start listen: %s", address+":"+port)

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

func getRandomPort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func savePortToCache(portFile string, port int) {
	err := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644)
	if err != nil {
		panic(err)
	}
}
