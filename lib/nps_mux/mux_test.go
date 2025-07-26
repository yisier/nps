package nps_mux

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"ehang.io/nps/lib/common"
)

var clientIp = "127.0.0.1" // 使用本地回环地址
var serverIp = "127.0.0.1" // 使用本地回环地址
var appIp = "127.0.0.1"    // 使用本地回环地址
var userIp = "127.0.0.1"   // 使用本地回环地址
var bridgePort = "9999"
var appPort = "9998"
var serverPort = "9997"
var appResultFileName = "app.txt"
var userResultFileName = "user.txt"
var serverResultFileName = "server.txt"
var clientResultFileName = "client.txt"
var dockerNetWorkName = "test"
var network = "172.18.0.0/16"
var fileSavePath = "/usr/src/myapp/"
var dataSize = 1024 * 1024 * 100

func TestMux(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(3)
	_ = createNetwork(dockerNetWorkName, network)
	go func() {
		defer wg.Done()
		_ = runDocker("server", dockerNetWorkName, serverIp, "TestServer", pwd)
	}()
	time.Sleep(time.Second * 5)
	go func() {
		defer wg.Done()
		_ = runDocker("client", dockerNetWorkName, clientIp, "TestClient", pwd)
	}()
	go func() {
		defer wg.Done()
		_ = runDocker("app", dockerNetWorkName, appIp, "TestApp", pwd)
	}()
	time.Sleep(time.Second * 5)
	_ = runDocker("user", dockerNetWorkName, userIp, "TestUser", pwd)
	wg.Wait()
	_ = deleteNetwork(dockerNetWorkName)
}

func writeResult(values []float64, outfile string) error {
	file, err := os.Create(fileSavePath + outfile)
	if err != nil {
		fmt.Println("writer", err)
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, v := range values {
		_, _ = writer.WriteString(fmt.Sprintf("%.2f", v))
		_, _ = writer.WriteString("\n")
		_ = writer.Flush()
	}
	return err
}

func appendResult(values []float64, outfile string) error {
	file, err := os.OpenFile(fileSavePath+outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("writer", err)
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, v := range values {
		_, _ = writer.WriteString(fmt.Sprintf("%.2f", v))
		_, _ = writer.WriteString("\n")
		_ = writer.Flush()
	}
	return err
}

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络配置的测试")
	}
	
	ip := net.ParseIP(serverIp)
	if ip == nil {
		t.Fatalf("not found interface %v", ip)
	}
	tc, err := NewTrafficControl(serverIp)
	if err != nil {
		t.Fatal(err, tc)
	}
	// do some tc settings
	// do some tc settings
	tc.delay("add", "50ms", "10ms", "10%")
	_ = tc.Run()
	_ = tc.bandwidth("1Mbit")
	// start bridge
	bridgeListener, err := net.Listen("tcp", serverIp+":"+bridgePort)
	if err != nil {
		t.Fatal(err)
	}
	// wait for client
	clientBridgeConn, err := bridgeListener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer clientBridgeConn.Close()
	// new mux
	mux := NewMux(clientBridgeConn, "tcp", 60)
	// start server port
	serverListener, err := net.Listen("tcp", serverIp+":"+serverPort)
	if err != nil {
		t.Fatal(err)
	}
	defer serverListener.Close()
	for {
		// accept user connection
		userConn, err := serverListener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		go func(userConn net.Conn) {
			// create a conn from mux
			clientConn, err := mux.NewConn()
			if err != nil {
				t.Fatal(err)
			}
			go io.Copy(userConn, clientConn)
			go func() {
				_ = writeResult([]float64{
					mux.bw.Get() / 1024 / 1024,
					math.Float64frombits(atomic.LoadUint64(&mux.latency)),
				}, serverResultFileName)
				ticker := time.NewTicker(time.Second * 1)
				for {
					select {
					case <-ticker.C:
						fmt.Println(mux.bw.Get()/1024/1024, math.Float64frombits(atomic.LoadUint64(&mux.latency)))
						_ = appendResult([]float64{
							mux.bw.Get() / 1024 / 1024,
							math.Float64frombits(atomic.LoadUint64(&mux.latency)),
						}, serverResultFileName)
					}
				}
			}()
			_, _ = io.Copy(clientConn, userConn)
			os.Exit(0)
		}(userConn)
	}
}
func TestClient(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络配置的测试")
	}
	
	ip := net.ParseIP(clientIp)
	if ip == nil {
		t.Fatalf("not found interface %v", ip)
	}
	// var addr = &net.TCPAddr{IP: ip, Port: common.TestClientPort} // 注释掉未使用的变量
	// start client
	clientListener, err := net.Listen("tcp", clientIp+":"+bridgePort)
	if err != nil {
		t.Fatal(err)
	}
	// wait for server
	serverBridgeConn, err := clientListener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer serverBridgeConn.Close()
	// new mux
	mux := NewMux(serverBridgeConn, "tcp", 60)
	// start accept user connection
	for {
		userConn, err := mux.Accept()
		if err != nil {
			t.Fatal(err)
		}
		go func(userConn net.Conn) {
			// connect to app
			appConn, err := net.Dial("tcp", appIp+":"+appPort)
			if err != nil {
				t.Fatal()
			}
			defer appConn.Close()
			defer userConn.Close()
			go io.Copy(userConn, appConn)
			go func() {
				_ = writeResult([]float64{
					mux.bw.Get() / 1024 / 1024,
					math.Float64frombits(atomic.LoadUint64(&mux.latency)),
				}, clientResultFileName)
				ticker := time.NewTicker(time.Second * 1)
				for {
					select {
					case <-ticker.C:
						_ = appendResult([]float64{
							mux.bw.Get() / 1024 / 1024,
							math.Float64frombits(atomic.LoadUint64(&mux.latency)),
						}, clientResultFileName)
					}
				}
			}()
			_, _ = io.Copy(appConn, userConn)
			os.Exit(0)
		}(userConn)
	}
}
func TestApp(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络配置的测试")
	}
	
	ip := net.ParseIP(appIp)
	if ip == nil {
		t.Fatalf("not found interface %v", ip)
	}
	
	// 创建流量控制对象
	tc, err := NewTrafficControl(appIp)
	if err != nil {
		t.Fatal(err)
	}
	
	// 配置延迟和带宽限制
	if err := tc.delay("add", "40ms", "5ms", "1%"); err != nil {
		t.Fatal("failed to set delay:", err)
	}
	
	if err := tc.bandwidth("1Mbit"); err != nil {
		t.Fatal("failed to set bandwidth limit:", err)
	}
	
	// 启动流量控制
	if err := tc.Run(); err != nil {
		t.Fatal("failed to start traffic control:", err)
	}
	
	// 启动应用监听
	appListener, err := net.Listen("tcp", appIp+":"+appPort)
	if err != nil {
		t.Fatal("failed to start application listener:", err)
	}
	for {
		userConn, err := appListener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		go func(userConn net.Conn) {
			b := bytes.Repeat([]byte{0}, 1024)
			startTime := time.Now()
			// send data to user
			for i := 0; i < dataSize/1024; i++ {
				n, err := userConn.Write(b)
				if err != nil {
					t.Fatal(err)
				}
				if n != 1024 {
					t.Fatal("the write len is not right")
				}
			}
			// send bandwidth
			writeBw := float64(dataSize/1024/1024) / time.Now().Sub(startTime).Seconds()
			// get 100md from user
			startTime = time.Now()
			readLen := 0
			for i := 0; i < 2<<32; i++ {
				n, err := userConn.Read(b)
				fmt.Println(n)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(readLen)
				readLen += n
				if readLen == dataSize {
					break
				}
			}
			if readLen != dataSize {
				t.Fatal("the read len is not right")
			}
			_, _ = userConn.Write([]byte{0})
			// read bandwidth
			readBw := float64(dataSize/1024/1024) / time.Now().Sub(startTime).Seconds()
			// save result
			err := writeResult([]float64{writeBw, readBw}, appResultFileName)
			if err != nil {
				t.Fatal(err)
			}
			os.Exit(0)
		}(userConn)
	}
}
func TestUser(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络配置的测试")
	}
	
	ip := net.ParseIP(userIp)
	if ip == nil {
		t.Fatalf("not found interface %v", ip)
	}
	// var addr = &net.TCPAddr{IP: ip, Port: common.TestUserPort} // 注释掉未使用的变量
	// start user
	userListener, err := net.Listen("tcp", userIp+":"+serverPort)
	if err != nil {
		t.Fatal(err)
	}
	b := bytes.Repeat([]byte{0}, 1024)
	startTime := time.Now()
	// get 100md from app
	readLen := 0
	for i := 0; i < 2<<32; i++ {
		n, err := userListener.Read(b)
		if err != nil {
			log.Fatal(err)
		}
		readLen += n
		if readLen == dataSize {
			break
		}
	}
	if readLen != dataSize {
		t.Fatal("the read len is not right", readLen, dataSize)
	}
	// read bandwidth
	readBw := float64(dataSize/1024/1024) / time.Now().Sub(startTime).Seconds()
	// send 100mb data to app
	startTime = time.Now()
	b = bytes.Repeat([]byte{0}, 1024)
	for i := 0; i < dataSize/1024; i++ {
		n, err := userListener.Write(b)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1024 {
			t.Fatal("the write len is not right")
		}
	}
	b = make([]byte, 1)
	_, err = io.ReadFull(userListener, b)
	if err != nil {
		t.Fatal(err)
	}
	// send bandwidth
	writeBw := float64(dataSize/1024/1024) / time.Now().Sub(startTime).Seconds()
	// save result
	err = writeResult([]float64{readBw, writeBw}, userResultFileName)
	if err != nil {
		t.Fatal(err)
	}
}
func TestNewMux2(t *testing.T) {
	tc, err := NewTrafficControl("")
	if err != nil {
		t.Fatal(err)
	}
	err = tc.RunNetRangeTest(func() {
		server(tc.Eth.EthAddr)
		client(tc.Eth.EthAddr)
		time.Sleep(time.Second * 3)
		rate := NewRate(1024 * 1024 * 3)
		rate.Start()
		conn2 = NewRateConn(rate, conn2)
		go func() {
			m2 := NewMux(conn2, "tcp", 60)
			for {
				c, err := m2.Accept()
				if err != nil {
					log.Println(err)
					continue
				}
				go func() {
					_, _ = c.Write(bytes.Repeat([]byte{0}, 1024*1024*100))
					_ = c.Close()
				}()
			}
		}()

		m1 := NewMux(conn1, "tcp", 60)
		tmpCpnn, err := m1.NewConn()
		if err != nil {
			log.Println("nps new conn err ", err)
			return
		}
		buf := make([]byte, 1024*1024)
		var count float64
		count = 0
		start := time.Now()
		for {
			n, err := tmpCpnn.Read(buf)
			count += float64(n)
			log.Println(m1.bw.Get())
			log.Println(uint32(math.Float64frombits(atomic.LoadUint64(&m1.latency))))
			if err != nil {
				log.Println(err)
				break
			}
		}
		log.Println("now rate", count/time.Now().Sub(start).Seconds()/1024/1024)
	})
	log.Println(err.Error())
}
func TestNewMux(t *testing.T) {
	go func() {
		_ = http.ListenAndServe("0.0.0.0:8889", nil)
	}()
	server("")
	client("")
	time.Sleep(time.Second * 3)
	go func() {
		m2 := NewMux(conn2, "tcp", 60)
		for {
			//log.Println("npc starting accept")
			c, err := m2.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			//log.Println("npc accept success ")
			c2, err := net.Dial("tcp", "127.0.0.1:80")
			if err != nil {
				log.Println(err)
				_ = c.Close()
				continue
			}
			//c2.(*net.TCPConn).SetReadBuffer(0)
			//c2.(*net.TCPConn).SetReadBuffer(0)
			go func(c2 net.Conn, c *conn) {
				go func() {
					buf := make([]byte, 32<<10)
					_, err = io.CopyBuffer(c2, c, buf)
					//if err != nil {
					//	log.Println("close npc by copy from nps", err, c.connId)
					//}
					_ = c2.Close()
					_ = c.Close()
				}()
				buf := make([]byte, 32<<10)
				_, err = io.CopyBuffer(c, c2, buf)
				//if err != nil {
				//	log.Println("close npc by copy from server", err, c.connId)
				//}
				_ = c2.Close()
				_ = c.Close()
			}(c2, c.(*conn))
		}
	}()

	go func() {
		m1 := NewMux(conn1, "tcp", 60)
		l, err := net.Listen("tcp", "127.0.0.1:7777")
		if err != nil {
			log.Println(err)
		}
		for {
			//log.Println("nps starting accept")
			conns, err := l.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			//conns.(*net.TCPConn).SetReadBuffer(0)
			//conns.(*net.TCPConn).SetReadBuffer(0)
			//log.Println("nps accept success starting New conn")
			tmpCpnn, err := m1.NewConn()
			if err != nil {
				log.Println("nps New conn err ", err)
				continue
			}
			//logs.Warn("nps New conn success ", tmpCpnn.connId)
			go func(tmpCpnn *conn, conns net.Conn) {
				go func() {
					buf := make([]byte, 32<<10)
					_, _ = io.CopyBuffer(tmpCpnn, conns, buf)
					//if err != nil {
					//	log.Println("close nps by copy from user", tmpCpnn.connId, err)
					//}
					_ = conns.Close()
					_ = tmpCpnn.Close()
				}()
				time.Sleep(time.Second)
				buf := make([]byte, 32<<10)
				_, err = io.CopyBuffer(conns, tmpCpnn, buf)
				//if err != nil {
				//	log.Println("close nps by copy from npc ", tmpCpnn.connId, err)
				//}
				_ = conns.Close()
				_ = tmpCpnn.Close()
			}(tmpCpnn, conns)
		}
	}()

	//go NewLogServer()
	time.Sleep(time.Second * 5)
	//for i := 0; i < 1; i++ {
	//	go test_raw(i)
	//}
	//test_request()

	for {
		time.Sleep(time.Second * 5)
	}
}

func server(ip string) {
	if ip == "" {
		ip = "127.0.0.1"
	}
	var err error
	l, err := net.Listen("tcp", ip+":9999")
	if err != nil {
		log.Println(err)
	}
	go func() {
		conn1, err = l.Accept()
		if err != nil {
			log.Println(err)
		}
	}()
	return
}

func client(ip string) {
	if ip == "" {
		ip = "127.0.0.1"
	}
	var err error
	conn2, err = net.Dial("tcp", ip+":9999")
	if err != nil {
		log.Println(err)
	}
}

func test_request() {
	conn, _ := net.Dial("tcp", "127.0.0.1:7777")
	for i := 0; i < 1000; i++ {
		_, _ = conn.Write([]byte(`GET / HTTP/1.1
Host: 127.0.0.1:7777
Connection: keep-alive


`))
		r, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			log.Println("close by read response err", err)
			break
		}
		log.Println("read response success", r)
		b, err := httputil.DumpResponse(r, true)
		if err != nil {
			log.Println("close by dump response err", err)
			break
		}
		fmt.Println(string(b[:20]), err)
		//time.Sleep(time.Second)
	}
	log.Println("finish")
}

func test_raw(k int) {
	for i := 0; i < 1000; i++ {
		ti := time.Now()
		conn, err := net.Dial("tcp", "127.0.0.1:7777")
		if err != nil {
			log.Println("conn dial err", err)
		}
		tid := time.Now()
		_, _ = conn.Write([]byte(`GET /videojs5/video.js HTTP/1.1
Host: 127.0.0.1:7777


`))
		tiw := time.Now()
		buf := make([]byte, 3572)
		n, err := io.ReadFull(conn, buf)
		//n, err := conn.Read(buf)
		if err != nil {
			log.Println("close by read response err", err)
			break
		}
		log.Println(n, string(buf[:50]), "\n--------------\n", string(buf[n-50:n]))
		//time.Sleep(time.Second)
		err = conn.Close()
		if err != nil {
			log.Println("close conn err ", err)
		}
		now := time.Now()
		du := now.Sub(ti).Seconds()
		dud := now.Sub(tid).Seconds()
		duw := now.Sub(tiw).Seconds()
		if du > 1 {
			log.Println("duration long", du, dud, duw, k, i)
		}
		if n != 3572 {
			log.Println("n loss", n, string(buf))
		}
	}
	log.Println("finish")
}

func TestNewConn(t *testing.T) {
	buf := make([]byte, 1024)
	log.Println(len(buf), cap(buf))
	//b := pool.GetBufPoolCopy()
	//b[0] = 1
	//b[1] = 2
	//b[2] = 3
	b := []byte{1, 2, 3}
	log.Println(copy(buf[:3], b), len(buf), cap(buf))
	log.Println(len(buf), buf[0])
}

func TestDQueue(t *testing.T) {
	d := new(bufDequeue)
	d.vals = make([]unsafe.Pointer, 8)
	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 10; i++ {
			log.Println(i)
			log.Println(d.popTail())
		}
	}()
	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 10; i++ {
			data := "test"
			go log.Println(i, unsafe.Pointer(&data), d.pushHead(unsafe.Pointer(&data)))
		}
	}()
	time.Sleep(time.Second * 3)
}

func TestChain(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:8889", nil))
	}()
	time.Sleep(time.Second * 5)
	d := new(bufChain)
	d.new(256)
	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 30000; i++ {
			unsa, ok := d.popTail()
			str := (*string)(unsa)
			if ok {
				fmt.Println(i, str, *str, ok)
				//logs.Warn(i, str, *str, ok)
			} else {
				fmt.Println("nil", i, ok)
				//logs.Warn("nil", i, ok)
			}
		}
	}()
	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 3000; i++ {
			go func(i int) {
				for n := 0; n < 10; n++ {
					data := "test " + strconv.Itoa(i) + strconv.Itoa(n)
					fmt.Println(data, unsafe.Pointer(&data))
					//logs.Warn(data, unsafe.Pointer(&data))
					d.pushHead(unsafe.Pointer(&data))
				}
			}(i)
		}
	}()
	time.Sleep(time.Second * 100000)
}

//func TestReceive(t *testing.T) {
//	go func() {
//		log.Println(http.ListenAndServe("0.0.0.0:8889", nil))
//	}()
//	logs.EnableFuncCallDepth(true)
//	logs.SetLogFuncCallDepth(3)
//	time.Sleep(time.Second * 5)
//	mux := New(Mux)
//	mux.bw.readBandwidth = float64(1*1024*1024)
//	mux.latency = float64(1/1000)
//	wind := New(receiveWindow)
//	wind.New(mux)
//	wind.
//	go func() {
//		time.Sleep(time.Second)
//		for i := 0; i < 36000; i++ {
//			data := d.Pop()
//			//fmt.Println(i, string(data.buf), err)
//			logs.Warn(i, string(data.content), data)
//		}
//	}()
//	go func() {
//		time.Sleep(time.Second*10)
//		for i := 0; i < 3000; i++ {
//			go func(i int) {
//				for n := 0; n < 10; n++{
//					data := New(common.muxPackager)
//					by := []byte("test " + strconv.Itoa(i) + strconv.Itoa(n))
//					_ = data.NewPac(common.MUX_NEW_MSG_PART, int32(i), by)
//					//fmt.Println(string((*data).buf), data)
//					logs.Warn(string((*data).content), data)
//					d.Push(data)
//				}
//			}(i)
//			go func(i int) {
//				data := New(common.muxPackager)
//				_ = data.NewPac(common.MUX_NEW_CONN, int32(i), nil)
//				//fmt.Println(string((*data).buf), data)
//				logs.Warn(data)
//				d.Push(data)
//			}(i)
//			go func(i int) {
//				data := New(common.muxPackager)
//				_ = data.NewPac(common.MUX_NEW_CONN_OK, int32(i), nil)
//				//fmt.Println(string((*data).buf), data)
//				logs.Warn(data)
//				d.Push(data)
//			}(i)
//		}
//	}()
//	time.Sleep(time.Second * 100000)
//}
