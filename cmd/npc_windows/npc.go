package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-autostart"
	"github.com/cnlh/nps/lib/common"
	"github.com/cnlh/nps/lib/daemon"
	"github.com/cnlh/nps/lib/version"
	"github.com/cnlh/nps/vender/github.com/astaxie/beego/logs"
	"github.com/getlantern/systray"
	ps "github.com/hanxi/go-powershell"
	"github.com/hanxi/go-powershell/backend"
	"github.com/hanxi/nps/client"
	"github.com/hanxi/nps/cmd/npc_windows/icon"
	"github.com/monochromegane/conflag"
)

var (
	serverAddr string
	verifyKey  string
	logType    string
	connType   string
	proxyURL   string
	logLevel   string
	logPath    string
)

var confFile = "npc.toml"
var flags *flag.FlagSet

func updateTips(serverAddr string, verifyKey string) {
	tips := fmt.Sprintf("server='%s'\nvkey='%s'", serverAddr, verifyKey)
	systray.SetTooltip(tips)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("npc")
	updateTips(serverAddr, verifyKey)

	mChecked := systray.AddMenuItem("Auto Startup", "Auto Startup npc on boot")
	filename := os.Args[0] // get command line first parameter
	app := &autostart.App{
		Name:        "npc",
		DisplayName: "npc",
		Exec:        []string{filename},
	}
	if app.IsEnabled() {
		mChecked.Check()
	}

	mEditConfig := systray.AddMenuItem("EditConfig", "Edit npc Config")

	go func() {
		for {
			select {
			case <-mChecked.ClickedCh:
				if mChecked.Checked() {
					if err := app.Disable(); err != nil {
						logs.Error("Disable Autostart Failed.")
					} else {
						mChecked.Uncheck()
					}
				} else {
					if err := app.Enable(); err != nil {
						logs.Error("Enable Autostart Failed.")
					} else {
						mChecked.Check()
					}
				}
			case <-mEditConfig.ClickedCh:
				{
					configLine, ok := inputBox("Input server and vkey", "Like this: -server=home.hanxi.info:2888 -vkey=pu74elp8h3v7ysaw", "")
					logs.Info(configLine)
					if ok {
						err := flags.Parse(strings.Split(configLine, " "))
						if err != nil {
							logs.Error("input error:%s", err.Error())
							return
						}
						writeConf()
					}
				}
			}
		}
	}()

	mQuit := systray.AddMenuItem("Quit", "Quit npc")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

	go start()

}

func onExit() {
	// clean up here
}

func main() {
	flags = getFlags()
	systray.Run(onReady, onExit)
}

func getConfPath() (string, error) {
	file, err := os.Open(confFile)
	defer func() {
		file.Close()
	}()

	if os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(confFile), os.ModePerm)
		s := "#server='xx.com:8080'\r\n#vkey='xxxx'\r\n"
		ioutil.WriteFile(confFile, []byte(s), os.ModePerm)
	}

	return confFile, nil
}

func getFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("npc", flag.ContinueOnError)
	flags.StringVar(&serverAddr, "server", "", "Server addr (ip:port)")
	flags.StringVar(&verifyKey, "vkey", "", "Authentication key")
	flags.StringVar(&logType, "log", "stdout", "Log output mode（stdout|file）")
	flags.StringVar(&connType, "type", "tcp", "Connection type with the server（kcp|tcp）")
	flags.StringVar(&logLevel, "log_level", "7", "log level 0~7")
	flags.StringVar(&logPath, "log_path", "npc.log", "npc log path")
	return flags
}

func reloadConf() (string, string) {
	var (
		server string
		vkey   string
	)
	flags := flag.NewFlagSet("npc-runtime", flag.ContinueOnError)
	flags.StringVar(&server, "server", "", "Server addr (ip:port)")
	flags.StringVar(&vkey, "vkey", "", "Authentication key")

	confPath, err := getConfPath()
	if err == nil {
		if confArgs, err := conflag.ArgsFrom(confPath); err == nil {
			flags.Parse(confArgs)
		} else {
			logs.Info("parse error:%s", err.Error())
		}
	}
	return server, vkey
}

func start() {
	if len(os.Args) > 1 {
		err := flags.Parse(os.Args[1:])
		if err != nil {
			logs.Error("args error:%s", err.Error())
			return
		}
	}

	daemon.InitDaemon("npc", common.GetRunPath(), common.GetTmpPath())
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	if logType == "stdout" {
		logs.SetLogger(logs.AdapterConsole, `{"level":`+logLevel+`,"color":true}`)
	} else {
		logs.SetLogger(logs.AdapterFile, `{"level":`+logLevel+`,"filename":"`+logPath+`","daily":false,"maxlines":100000,"color":true}`)
	}

	logs.Info("the version of client is %s, the core version of client is %s", version.VERSION, version.GetVersion())
	go func() {
		for {
			server, vkey := reloadConf()
			updateTips(server, vkey)
			logs.Info("server:%s, vkey:%s", server, vkey)
			client.NewRPClient(server, vkey, connType, proxyURL, nil).Start()
			logs.Info("It will be reconnected in five seconds")
			time.Sleep(time.Second * 5)
		}
	}()
}

// inputBox displays a dialog box, returning the entered value and a bool for success
func inputBox(title, message, defaultAnswer string) (string, bool) {
	shell, err := ps.New(&backend.Local{})
	if err != nil {
		panic(err)
	}
	defer shell.Exit()

	out, _, err := shell.Execute(`
		[void][Reflection.Assembly]::LoadWithPartialName('Microsoft.VisualBasic')
		$title = '` + title + `'
		$msg = '` + message + `'
		$default = '` + defaultAnswer + `'
		$answer = [Microsoft.VisualBasic.Interaction]::InputBox($msg, $title, $default)
		Write-Output $answer
		`)
	// FIXME: if cancel button is pressed in dialog, we should return false
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func writeConf() {
	confPath, err := getConfPath()
	if err != nil {
		return
	}
	s := fmt.Sprintf("server='%s'\r\nvkey='%s'\r\n", serverAddr, verifyKey)
	ioutil.WriteFile(confPath, []byte(s), os.ModePerm)
}
