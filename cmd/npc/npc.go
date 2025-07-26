package main

import (
	"bufio"
	"ehang.io/nps/lib/crypt"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"ehang.io/nps/client"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/config"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/lib/install"
	"ehang.io/nps/lib/version"
	"github.com/astaxie/beego/logs"
	"github.com/ccding/go-stun/stun"
	"github.com/fatih/color"
	"github.com/kardianos/service"
)

var (
	serverAddr     = flag.String("server", "", "Server addr (ip:port)")
	configPath     = flag.String("config", "", "Configuration file path")
	verifyKey      = flag.String("vkey", "", "Authentication key")
	logType        = flag.String("log", "stdout", "Log output mode（stdout|file）")
	connType       = flag.String("type", "tcp", "Connection type with the server（kcp|tcp）")
	proxyUrl       = flag.String("proxy", "", "proxy socks5 url(eg:socks5://111:222@127.0.0.1:9007)")
	logLevel       = flag.String("log_level", "7", "log level 0~7")
	registerTime   = flag.Int("time", 2, "register time long /h")
	localPort      = flag.Int("local_port", 2000, "p2p local port")
	password       = flag.String("password", "", "p2p password flag")
	target         = flag.String("target", "", "p2p target")
	localType      = flag.String("local_type", "p2p", "p2p target")
	logPath        = flag.String("log_path", "", "npc log path")
	debug          = flag.Bool("debug", true, "npc debug")
	pprofAddr      = flag.String("pprof", "", "PProf debug addr (ip:port)")
	stunAddr       = flag.String("stun_addr", "stun.stunprotocol.org:3478", "stun server address (eg:stun.stunprotocol.org:3478)")
	ver            = flag.Bool("version", false, "show current version")
	disconnectTime = flag.Int("disconnect_timeout", 60, "not receiving check packet times, until timeout will disconnect the client")
	tlsEnable      = flag.Bool("tls_enable", false, "enable tls")
)

func main() {
	flag.Parse()
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	if *ver {
		common.PrintVersion()
		return
	}
	if *logPath == "" {
		*logPath = common.GetNpcLogPath()
	}
	if common.IsWindows() {
		*logPath = strings.Replace(*logPath, "\\", "\\\\", -1)
	}
	if *debug {
		logs.SetLogger(logs.AdapterConsole, `{"level":`+*logLevel+`,"color":true}`)
	} else {
		logs.SetLogger(logs.AdapterFile, `{"level":`+*logLevel+`,"filename":"`+*logPath+`","daily":false,"maxlines":100000,"color":true}`)
	}

	// init service
	options := make(service.KeyValue)
	svcConfig := &service.Config{
		Name:        "Npc",
		DisplayName: "nps内网穿透客户端",
		Description: "一款轻量级、功能强大的内网穿透代理服务器。支持tcp、udp流量转发，支持内网http代理、内网socks5代理，同时支持snappy压缩、站点保护、加密传输、多路复用、header修改等。支持web图形化管理，集成多用户模式。",
		Option:      options,
	}
	if !common.IsWindows() {
		svcConfig.Dependencies = []string{
			"Requires=network.target",
			"After=network-online.target syslog.target"}
		svcConfig.Option["SystemdScript"] = install.SystemdScript
		svcConfig.Option["SysvScript"] = install.SysvScript
	}
	for _, v := range os.Args[1:] {
		switch v {
		case "install", "start", "stop", "uninstall", "restart":
			continue
		}
		if !strings.Contains(v, "-service=") && !strings.Contains(v, "-debug=") {
			svcConfig.Arguments = append(svcConfig.Arguments, v)
		}
	}
	svcConfig.Arguments = append(svcConfig.Arguments, "-debug=false")
	prg := &npc{
		exit: make(chan struct{}),
	}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logs.Error(err, "service function disabled")
		run()
		// run without service
		wg := sync.WaitGroup{}
		wg.Add(1)
		wg.Wait()
		return
	}
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "status":
			if len(os.Args) > 2 {
				path := strings.Replace(os.Args[2], "-config=", "", -1)
				client.GetTaskStatus(path)
			}
		case "register":
			flag.CommandLine.Parse(os.Args[2:])
			client.RegisterLocalIp(*serverAddr, *verifyKey, *connType, *proxyUrl, *registerTime)
		case "update":
			install.UpdateNpc()
			return
		case "nat":
			c := stun.NewClient()
			flag.CommandLine.Parse(os.Args[2:])
			c.SetServerAddr(*stunAddr)
			nat, host, err := c.Discover()
			if err != nil || host == nil {
				logs.Error("get nat type error", err)
				return
			}
			fmt.Printf("nat type: %s \npublic address: %s\n", nat.String(), host.String())
			os.Exit(0)
		case "start", "stop", "restart":
			// support busyBox and sysV, for openWrt
			if service.Platform() == "unix-systemv" {
				logs.Info("unix-systemv service")
				cmd := exec.Command("/etc/init.d/"+svcConfig.Name, os.Args[1])
				err := cmd.Run()
				if err != nil {
					logs.Error(err)
				}
				return
			}
			err := service.Control(s, os.Args[1])
			if err != nil {
				logs.Error("Valid actions: %q\n%s", service.ControlAction, err.Error())
			}
			return
		case "install":
			service.Control(s, "stop")
			service.Control(s, "uninstall")
			install.InstallNpc()
			err := service.Control(s, os.Args[1])
			if err != nil {
				logs.Error("Valid actions: %q\n%s", service.ControlAction, err.Error())
			}
			if service.Platform() == "unix-systemv" {
				logs.Info("unix-systemv service")
				confPath := "/etc/init.d/" + svcConfig.Name
				os.Symlink(confPath, "/etc/rc.d/S90"+svcConfig.Name)
				os.Symlink(confPath, "/etc/rc.d/K02"+svcConfig.Name)
			}
			return
		case "uninstall":
			err := service.Control(s, os.Args[1])
			if err != nil {
				logs.Error("Valid actions: %q\n%s", service.ControlAction, err.Error())
			}
			if service.Platform() == "unix-systemv" {
				logs.Info("unix-systemv service")
				os.Remove("/etc/rc.d/S90" + svcConfig.Name)
				os.Remove("/etc/rc.d/K02" + svcConfig.Name)
			}
			return
		}
	}
	s.Run()
}

type npc struct {
	exit chan struct{}
}

func (p *npc) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *npc) Stop(s service.Service) error {
	close(p.exit)
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}

func (p *npc) run() error {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logs.Warning("npc: panic serving %v: %v\n%s", err, string(buf))
		}
	}()
	run()
	select {
	case <-p.exit:
		logs.Warning("stop...")
	}
	return nil
}

func run() {
	common.InitPProfFromArg(*pprofAddr)
	//p2p or secret command
	if *password != "" {
		commonConfig := new(config.CommonConfig)
		commonConfig.Server = *serverAddr
		commonConfig.VKey = *verifyKey
		commonConfig.Tp = *connType
		localServer := new(config.LocalServer)
		localServer.Type = *localType
		localServer.Password = *password
		localServer.Target = *target
		localServer.Port = *localPort
		commonConfig.Client = new(file.Client)
		commonConfig.Client.Cnf = new(file.Config)
		go client.StartLocalServer(localServer, commonConfig)
		return
	}
	env := common.GetEnvMap()
	if *serverAddr == "" {
		*serverAddr, _ = env["NPC_SERVER_ADDR"]
	}
	if *verifyKey == "" {
		*verifyKey, _ = env["NPC_SERVER_VKEY"]
	}
	if *verifyKey != "" && *serverAddr != "" && *configPath == "" {
		client.SetTlsEnable(*tlsEnable)
		logs.Info("the version of client is %s, the core version of client is %s,tls enable is %t", version.VERSION, version.GetVersion(), client.GetTlsEnable())

		vkeys := strings.Split(*verifyKey, `,`)
		for _, key := range vkeys {
			key := key
			go func() {
				for {
					logs.Info("start vkey:" + key)
					client.NewRPClient(*serverAddr, key, *connType, *proxyUrl, nil, *disconnectTime).Start()
					logs.Info("Client closed! It will be reconnected in five seconds")
					time.Sleep(time.Second * 5)
				}
			}()
		}

	} else {
		if *configPath == "" {
			*configPath = common.GetConfigPath()
		}

		// 判断路径下是否有配置文件
		if common.FileExists(*configPath) {
			logs.Info("配置文件模式启动")
			go client.StartFromFile(*configPath)
		} else {
			// 无配置文件模式双击运行
			printSlogan()
			inputCmd()
		}
	}
}

func printSlogan() {
	green := color.New(color.FgGreen).SprintFunc()
	// 第一次输入，如果输入 1,2,3，4 则需要输入秘钥，否则

	fmt.Printf("%s", green(""))

	fmt.Printf("\033[32;0m###########################################################\n")
	fmt.Printf("\033[32;0m#                   \033[31mNPS内网穿透客户端\033[0m                     #\n")
	fmt.Printf("\033[32;0m#                            			          #\n")
	fmt.Printf("\033[32;0m#\033[32m 地址：\033[31;0mhttps://github.com/yisier/nps\033[0m                     #\n")
	fmt.Printf("\033[32;0m#\033[32m 提示：\033[32;0m1、涉及到系统服务的需要以管理员身份运行\033[0m\033[32;0m	          #\n")
	fmt.Printf("\033[32;0m#\033[32m       \033[32;0m2、直接启动或[注册系统服务]需要使用[快捷启动命令]\033[0m\033[32;0m #\n")
	fmt.Printf("\033[32;0m#\033[32m       \033[32;0m3、其他命令如卸载/启动/停止只需要输入[vkey]\033[0m\033[32;0m	  #\n")
	fmt.Printf("\033[32;0m###########################################################\n")
	fmt.Printf("\033[0m") // 重置颜色

	fmt.Printf("\n")

	fmt.Printf("\u001B[32m输入[1]\u001B[0m - 注册系统服务\n")
	fmt.Printf("\u001B[32m输入[2]\u001B[0m - 卸载系统服务\n")
	fmt.Printf("---------------------\n")
	fmt.Printf("\u001B[32m输入[3]\u001B[0m - 启动系统服务\n")
	fmt.Printf("\u001B[32m输入[4]\u001B[0m - 停止系统服务\n")
	fmt.Printf("---------------------\n")
	fmt.Printf("\u001B[32m输入[0]\u001B[0m - 退出\n")
	fmt.Printf("---------------------\n")
	fmt.Printf("直接输入[快捷启动命令]则是启动隧道,多个[快捷启动命令]用英文逗号拼接\n")
	fmt.Printf("\n")
}

func inputCmd() {

	var flag string
	fmt.Printf("请输入：")

	stdin := bufio.NewReader(os.Stdin)
	_, err := fmt.Fscanln(stdin, &flag)
	if err != nil {
		fmt.Println("输入有误")
	} else {
		if flag == "0" {
			os.Exit(0)
		}

		flag := strings.Replace(flag, " ", "", -1)

		// 如果输入不等于 1,2,3，4，则启动隧道
		if flag != "1" && flag != "2" && flag != "3" && flag != "4" {

			vkeys := strings.Split(flag, `,`)
			var cmdArray []string

			for _, key := range vkeys {
				startCmd, err := crypt.Base64Decoding(key)
				if err != nil {
					fmt.Println("快捷启动命令解析失败")
					inputCmd()
					return
				}

				cmdArray = append(cmdArray, startCmd)
			}

			for _, item := range cmdArray {
				startNpcServer(item)
			}

		} else {
			systemService(flag)
		}
	}
}

func startNpcServer(startCmd string) {
	var serAddr string
	var vkey string
	var tls string
	array := strings.Fields(startCmd)
	serAddr = array[0]
	vkey = array[1]
	if len(array) > 2 {
		tls = array[2]
	}
	go func() {
		for {
			if tls == "-tls_enable=true" {
				client.SetTlsEnable(true)
				logs.Info("start cmd:-server=" + serAddr + " -vkey=" + vkey + " " + tls)
				logs.Info("the version of client is %s, the core version of client is %s,tls enable is %t", version.VERSION, version.GetVersion(), client.GetTlsEnable())
			} else {
				client.SetTlsEnable(false)
				logs.Info("start cmd:-server=" + serAddr + " -vkey=" + vkey)
				logs.Info("the version of client is %s, the core version of client is %s", version.VERSION, version.GetVersion())
			}

			client.NewRPClient(serAddr, vkey, *connType, *proxyUrl, nil, *disconnectTime).Start()
			logs.Info("Client closed! It will be reconnected in five seconds")
			time.Sleep(time.Second * 5)
		}
	}()
}

func systemService(flag string) {

	if flag == "1" {
		fmt.Printf("请输入[快捷启动命令],多个[快捷启动命令]用英文逗号拼接：")
	} else {
		fmt.Printf("请输入[VKEY],多个[VKEY]用英文逗号拼接：")
	}

	var vkey string
	stdin := bufio.NewReader(os.Stdin)
	_, err := fmt.Fscanln(stdin, &vkey)

	if err != nil {
		fmt.Println("输入错误，请重试")
		systemService(flag)
		return
	} else {
		if vkey == "0" {
			os.Exit(0)
		}
	}

	vkey = strings.Replace(vkey, " ", "", -1)

	vkeys := strings.Split(vkey, `,`)

	if flag == "1" {
		var cmdArray []string
		for _, key := range vkeys {
			startCmd, err := crypt.Base64Decoding(key)
			if err != nil {
				fmt.Println("快捷启动命令解析失败")
				systemService(flag)
				return
			}
			cmdArray = append(cmdArray, startCmd)
		}

		for _, item := range cmdArray {
			array := strings.Fields(item)
			systemPro(flag, array[0], array[1])
		}
	} else {
		for _, key := range vkeys {
			systemPro(flag, "", key)
		}

	}

	inputCmd()
	return
}

func systemPro(flag string, serAddr string, vkey string) {
	// init service
	prg := &npc{
		exit: make(chan struct{}),
	}
	options := make(service.KeyValue)
	svcConfig := &service.Config{
		Name:        "nps-client-" + vkey,
		DisplayName: "nps-client-" + vkey,
		Description: "NPS内网穿透客户端，支持tcp、udp流量转发，支持内网http代理，地址：https://github.com/yisier/nps",
		Option:      options,
	}
	s, _ := service.New(prg, svcConfig)

	switch flag {
	case "1":
		svcConfig.Arguments = append(svcConfig.Arguments, "-server="+serAddr)
		svcConfig.Arguments = append(svcConfig.Arguments, "-vkey="+vkey)
		svcConfig.Arguments = append(svcConfig.Arguments, "-debug=false")

		*logPath = common.GetNpcLogPath()
		if common.IsWindows() {
			*logPath = strings.Replace(*logPath, "\\", "\\\\", -1)
		}

		*logPath = strings.Replace(*logPath, "npc.log", "npc-"+vkey+".log", -1)
		svcConfig.Arguments = append(svcConfig.Arguments, "-log_path="+*logPath)

		logs.NewLogger()
		logs.Reset()
		logs.EnableFuncCallDepth(true)
		logs.SetLogFuncCallDepth(3)
		logs.SetLogger(logs.AdapterFile, `{"level":`+*logLevel+`,"filename":"`+*logPath+`","daily":false,"maxlines":100000,"color":true}`)

		install.InstallNpc()
		err := service.Control(s, "install")
		if err != nil {
			fmt.Println("隧道["+vkey+"]安装到系统服务失败", err)
			return
		} else {
			fmt.Println("隧道[" + vkey + "]已经安装到系统")
		}
		if service.Platform() == "unix-systemv" {
			logs.Info("unix-systemv service")
			confPath := "/etc/init.d/" + svcConfig.Name
			os.Symlink(confPath, "/etc/rc.d/S90"+svcConfig.Name)
			os.Symlink(confPath, "/etc/rc.d/K02"+svcConfig.Name)
		}

		err2 := service.Control(s, "start")
		if err2 != nil {
			fmt.Println("隧道["+vkey+"]启动服务失败", err2)
		} else {
			fmt.Println("隧道[" + vkey + "]服务已启动")
		}

		return
	case "2":
		// 卸载系统服务
		err := service.Control(s, "stop")
		if err != nil {
			fmt.Println("隧道["+vkey+"]服务停止失败", err)
		} else {
			fmt.Println("隧道[" + vkey + "]服务已停止")
		}

		err = service.Control(s, "uninstall")
		if err != nil {
			fmt.Println("隧道["+vkey+"]服务卸载失败", err)
		}
		if service.Platform() == "unix-systemv" {
			fmt.Println("unix-systemv service")
			os.Remove("/etc/rc.d/S90" + svcConfig.Name)
			os.Remove("/etc/rc.d/K02" + svcConfig.Name)
		}

		if err == nil {
			fmt.Println("隧道[" + vkey + "]服务已卸载成功")
		}

		return

	case "3":
		//启动系统服务
		if service.Platform() == "unix-systemv" {
			logs.Info("unix-systemv service")
			cmd := exec.Command("/etc/init.d/"+svcConfig.Name, "start")
			err := cmd.Run()
			if err != nil {
				logs.Error(err)
			}
			return
		}
		err := service.Control(s, "start")
		if err != nil {
			fmt.Println("隧道["+vkey+"]服务启动失败", err)
		} else {
			fmt.Println("隧道[" + vkey + "]服务启动成功")
		}

		return
	case "4":
		if service.Platform() == "unix-systemv" {
			logs.Info("unix-systemv service")
			cmd := exec.Command("/etc/init.d/"+svcConfig.Name, "stop")
			err := cmd.Run()
			if err != nil {
				logs.Error(err)
			}
			return
		}
		err := service.Control(s, "stop")
		if err != nil {
			fmt.Println("隧道["+vkey+"]服务停止失败", err)
		} else {
			fmt.Println("隧道[" + vkey + "]服务停止成功")
		}

		return
	}
}
