package tool

import (
	"math"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ehang.io/nps/lib/common"
	"github.com/astaxie/beego"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var (
	ports          []int
	ServerStatus   []map[string]interface{}
	ServerStatusMu sync.RWMutex
	IORateCache    atomic.Value
)

func StartSystemInfo() {
	if b, err := beego.AppConfig.Bool("system_info_display"); err == nil && b {
		ServerStatus = make([]map[string]interface{}, 0, 1500)
		go getSeverStatus()
	}
}

func InitAllowPort() {
	p := beego.AppConfig.String("allow_ports")
	ports = common.GetPorts(p)
}

func TestServerPort(p int, m string) (b bool) {
	if m == "p2p" || m == "secret" {
		return true
	}
	if p > 65535 || p < 0 {
		return false
	}
	if len(ports) != 0 {
		if !common.InIntArr(ports, p) {
			return false
		}
	}
	if m == "udp" {
		b = common.TestUdpPort(p)
	} else {
		b = common.TestTcpPort(p)
	}
	return
}

func GenerateServerPort(m string) int {
	for i := 0; i < 1000; i++ {
		//生成随机数 1024 - 65535
		serverPort := rand.Intn(65535)
		if serverPort < 1024 {
			serverPort = 1024
		}

		if TestServerPort(serverPort, m) {
			return serverPort
		}
	}
	return 0 // 超过最大重试次数，返回0表示无法分配端口
}

func getSeverStatus() {
	for {
		if len(ServerStatus) < 10 {
			time.Sleep(time.Second)
		} else {
			time.Sleep(time.Minute)
		}
		cpuPercet, _ := cpu.Percent(0, true)
		var cpuAll float64
		for _, v := range cpuPercet {
			cpuAll += v
		}
		m := make(map[string]interface{})
		loads, _ := load.Avg()
		m["load1"] = loads.Load1
		m["load5"] = loads.Load5
		m["load15"] = loads.Load15
		m["cpu"] = math.Round(cpuAll / float64(len(cpuPercet)))
		swap, _ := mem.SwapMemory()
		m["swap_mem"] = math.Round(swap.UsedPercent)
		vir, _ := mem.VirtualMemory()
		m["virtual_mem"] = math.Round(vir.UsedPercent)
		conn, _ := net.ProtoCounters(nil)
		if cached, ok := IORateCache.Load().(map[string]uint64); ok {
			m["io_send"] = cached["io_send"]
			m["io_recv"] = cached["io_recv"]
		}
		t := time.Now()
		m["time"] = strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + ":" + strconv.Itoa(t.Second())

		for _, v := range conn {
			m[v.Protocol] = v.Stats["CurrEstab"]
		}
		if len(ServerStatus) >= 1440 {
			ServerStatusMu.Lock()
			ServerStatus = ServerStatus[1:]
			ServerStatusMu.Unlock()
		}
		ServerStatusMu.Lock()
		ServerStatus = append(ServerStatus, m)
		ServerStatusMu.Unlock()
	}
}

// StartIORateCollector 启动后台 IO 速率采集，每 2s 刷新一次 IORateCache
func StartIORateCollector() {
	go collectIORate()
}

func collectIORate() {
	var lastSent, lastRecv uint64
	// 修复：无论首次 IOCounters 是否成功，都把 lastTime 设为 now，
	// 否则首次有效采样的 elapsed 会是几十年，导致 sendRate/recvRate 直接为 0
	lastTime := time.Now()
	if io, err := net.IOCounters(false); err == nil && len(io) > 0 {
		lastSent = io[0].BytesSent
		lastRecv = io[0].BytesRecv
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		io, err := net.IOCounters(false)
		if err != nil || len(io) == 0 {
			continue
		}
		elapsed := now.Sub(lastTime).Seconds()
		if elapsed <= 0 {
			continue
		}
		var sendRate, recvRate uint64
		if io[0].BytesSent >= lastSent {
			sendRate = uint64(float64(io[0].BytesSent-lastSent) / elapsed)
		}
		if io[0].BytesRecv >= lastRecv {
			recvRate = uint64(float64(io[0].BytesRecv-lastRecv) / elapsed)
		}
		lastSent = io[0].BytesSent
		lastRecv = io[0].BytesRecv
		lastTime = now
		IORateCache.Store(map[string]uint64{
			"io_send": sendRate,
			"io_recv": recvRate,
		})
	}
}
