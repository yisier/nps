package common

import (
	"bytes"
	"ehang.io/nps/lib/version"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ehang.io/nps/lib/crypt"
)

// Get the corresponding IP address through domain name
func GetHostByName(hostname string) string {
	if !DomainCheck(hostname) {
		return hostname
	}
	ips, _ := net.LookupIP(hostname)
	if ips != nil {
		for _, v := range ips {
			if v.To4() != nil {
				return v.String()
			}
		}
	}
	return ""
}

// Check the legality of domain
func DomainCheck(domain string) bool {
	var match bool
	IsLine := "^((http://)|(https://))?([a-zA-Z0-9]([a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])?\\.)+[a-zA-Z]{2,6}(/)"
	NotLine := "^((http://)|(https://))?([a-zA-Z0-9]([a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])?\\.)+[a-zA-Z]{2,6}"
	match, _ = regexp.MatchString(IsLine, domain)
	if !match {
		match, _ = regexp.MatchString(NotLine, domain)
	}
	return match
}

// Check if the Request request is validated
func CheckAuth(r *http.Request, user, passwd string) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		s = strings.SplitN(r.Header.Get("Proxy-Authorization"), " ", 2)
		if len(s) != 2 {
			return false
		}
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}
	return pair[0] == user && pair[1] == passwd
}

// get bool by str
func GetBoolByStr(s string) bool {
	switch s {
	case "1", "true":
		return true
	}
	return false
}

// get str by bool
func GetStrByBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// int
func GetIntNoErrByStr(str string) int {
	i, _ := strconv.Atoi(strings.TrimSpace(str))
	return i
}

// Get verify value
func Getverifyval(vkey string) string {
	return crypt.Md5(vkey)
}

// Change headers and host of request
func ChangeHostAndHeader(r *http.Request, host string, header string, addr string) {
	if host != "" {
		r.Host = host
	}
	if header != "" {
		h := strings.Split(header, "\n")
		for _, v := range h {
			hd := strings.Split(v, ":")
			if len(hd) == 2 {
				r.Header.Set(hd[0], hd[1])
			}
		}
	}
	addr = strings.Split(addr, ":")[0]
	if prior, ok := r.Header["X-Forwarded-For"]; ok {
		addr = strings.Join(prior, ", ") + ", " + addr
	}

	addOrigin, _ := beego.AppConfig.Bool("http_add_origin_header")

	if addOrigin {
		logs.Debug("set X-Forwarded-For X-Real-IP = " + addr)
		r.Header.Set("X-Forwarded-For", addr)
		r.Header.Set("X-Real-IP", addr)
	}
}

// Read file content by file path
func ReadAllFromFile(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Judge whether the TCP port can open normally
func TestTcpPort(port int) bool {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP("0.0.0.0"), port, ""})
	defer func() {
		if l != nil {
			l.Close()
		}
	}()
	if err != nil {
		return false
	}
	return true
}

// Judge whether the UDP port can open normally
func TestUdpPort(port int) bool {
	l, err := net.ListenUDP("udp", &net.UDPAddr{net.ParseIP("0.0.0.0"), port, ""})
	defer func() {
		if l != nil {
			l.Close()
		}
	}()
	if err != nil {
		return false
	}
	return true
}

// Write length and individual byte data
// Length prevents sticking
// # Characters are used to separate data
func BinaryWrite(raw *bytes.Buffer, v ...string) {
	b := GetWriteStr(v...)
	binary.Write(raw, binary.LittleEndian, int32(len(b)))
	binary.Write(raw, binary.LittleEndian, b)
}

// get seq str
func GetWriteStr(v ...string) []byte {
	buffer := new(bytes.Buffer)
	var l int32
	for _, v := range v {
		l += int32(len([]byte(v))) + int32(len([]byte(CONN_DATA_SEQ)))
		binary.Write(buffer, binary.LittleEndian, []byte(v))
		binary.Write(buffer, binary.LittleEndian, []byte(CONN_DATA_SEQ))
	}
	return buffer.Bytes()
}

// inArray str interface
func InStrArr(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

// inArray int interface
func InIntArr(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

// format ports str to a int array
func GetPorts(p string) []int {
	var ps []int
	arr := strings.Split(p, ",")
	for _, v := range arr {
		fw := strings.Split(v, "-")
		if len(fw) == 2 {
			if IsPort(fw[0]) && IsPort(fw[1]) {
				start, _ := strconv.Atoi(fw[0])
				end, _ := strconv.Atoi(fw[1])
				for i := start; i <= end; i++ {
					ps = append(ps, i)
				}
			} else {
				continue
			}
		} else if IsPort(v) {
			p, _ := strconv.Atoi(v)
			ps = append(ps, p)
		}
	}
	return ps
}

// is the string a port
func IsPort(p string) bool {
	pi, err := strconv.Atoi(p)
	if err != nil {
		return false
	}
	if pi > 65536 || pi < 1 {
		return false
	}
	return true
}

// if the s is just a port,return 127.0.0.1:s
func FormatAddress(s string) string {
	if strings.Contains(s, ":") {
		return s
	}
	return "127.0.0.1:" + s
}

// get address from the complete address
func GetIpByAddr(addr string) string {
	// 如果是 IPv6 地址，可能的格式为[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080或[2001:0db8:85a3:0000:0000:8a2e:0370:7334]
	// IPv4 地址，可能的格式为192.168.1.1:80或192.168.1.1
	// 截取地址
	if strings.Contains(addr, "[") {
		addr = strings.Split(addr, "[")[1]
		addr = strings.Split(addr, "]")[0]
	} else {
		addr = strings.Split(addr, ":")[0]
	}
	return addr
}

// get port from the complete address
func GetPortByAddr(addr string) int {
	arr := strings.Split(addr, ":")
	if len(arr) < 2 {
		return 0
	}
	p, err := strconv.Atoi(arr[1])
	if err != nil {
		return 0
	}
	return p
}

func in(target string, str_array []string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

// 判断访问地址是否在黑名单内
func IsBlackIp(ipPort, vkey string, blackIpList []string) bool {
	ip := GetIpByAddr(ipPort)
	if in(ip, blackIpList) {
		logs.Error("IP地址[" + ip + "]在隧道[" + vkey + "]黑名单列表内")
		return true
	}

	return false
}

// 判断访问地址是否在白名单内
func IsAuthIp(ipPort, vkey string, authIpList []string) bool {
	ip := GetIpByAddr(ipPort)

	if in(ip, authIpList) {
		return false
	}
	logs.Error("IP地址[" + ip + "]不在隧道[" + vkey + "]白名单列表内")
	return true
}

func CopyBuffer(dst io.Writer, src io.Reader, label ...string) (written int64, err error) {
	buf := CopyBuff.Get()
	defer CopyBuff.Put(buf)
	for {
		nr, er := src.Read(buf)
		//if len(pr)>0 && pr[0] && nr > 50 {
		//	logs.Warn(string(buf[:50]))
		//}
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err
}

// send this ip forget to get a local udp port
func GetLocalUdpAddr() (net.Conn, error) {
	tmpConn, err := net.Dial("udp", "114.114.114.114:53")
	if err != nil {
		return nil, err
	}
	return tmpConn, tmpConn.Close()
}

// parse template
func ParseStr(str string) (string, error) {
	tmp := template.New("npc")
	var err error
	w := new(bytes.Buffer)
	if tmp, err = tmp.Parse(str); err != nil {
		return "", err
	}
	if err = tmp.Execute(w, GetEnvMap()); err != nil {
		return "", err
	}
	return w.String(), nil
}

// get env
func GetEnvMap() map[string]string {
	m := make(map[string]string)
	environ := os.Environ()
	for i := range environ {
		tmp := strings.Split(environ[i], "=")
		if len(tmp) == 2 {
			m[tmp[0]] = tmp[1]
		}
	}
	return m
}

// throw the empty element of the string array
func TrimArr(arr []string) []string {
	newArr := make([]string, 0)
	for _, v := range arr {
		if v != "" {
			newArr = append(newArr, v)
		}
	}
	return newArr
}

func IsArrContains(arr []string, val string) bool {
	if arr == nil {
		return false
	}
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

// remove value from string array
func RemoveArrVal(arr []string, val string) []string {
	for k, v := range arr {
		if v == val {
			arr = append(arr[:k], arr[k+1:]...)
			return arr
		}
	}
	return arr
}

// convert bytes to num
func BytesToNum(b []byte) int {
	var str string
	for i := 0; i < len(b); i++ {
		str += strconv.Itoa(int(b[i]))
	}
	x, _ := strconv.Atoi(str)
	return int(x)
}

// get the length of the sync map
func GeSynctMapLen(m sync.Map) int {
	var c int
	m.Range(func(key, value interface{}) bool {
		c++
		return true
	})
	return c
}

func GetExtFromPath(path string) string {
	s := strings.Split(path, ".")
	re, err := regexp.Compile(`(\w+)`)
	if err != nil {
		return ""
	}
	return string(re.Find([]byte(s[0])))
}

var externalIp string

// externalIpServices are queried in order until one returns a valid IP.
// HTTPS plain-text echo services are preferred, because a plain HTTP endpoint
// can be hijacked or transparently cached by some ISPs (see #355).
var externalIpServices = []string{
	"https://ifconfig.me/ip",
	"https://ipinfo.io/ip",
	"http://v4.ipip.net",
}

// GetExternalIp returns the public IP of this server by asking an external
// "what is my IP" service. The result is cached only after a valid answer,
// so a transient failure or a hijacked/empty response is not cached forever.
func GetExternalIp() string {
	if externalIp != "" {
		return externalIp
	}
	client := &http.Client{Timeout: 3 * time.Second}
	for _, url := range externalIpServices {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		content, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}
		ip := strings.TrimSpace(string(content))
		if net.ParseIP(ip) == nil {
			continue
		}
		externalIp = ip
		return externalIp
	}
	return ""
}

func GetIntranetIp() (error, string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, ""
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return nil, ipnet.IP.To4().String()
			}
		}
	}
	return errors.New("get intranet ip error"), ""
}

// isUsefulLocalIPv4 reports whether ip is a usable unicast IPv4 for display:
// excludes loopback, link-local (169.254.0.0/16), unspecified and multicast.
// No RFC1918 requirement, so non-standard private ranges used in practice
// (e.g. 192.169.0.0/16) are kept.
func isUsefulLocalIPv4(ip net.IP) bool {
	if ip == nil {
		return false
	}
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return !v4.IsLoopback() && !v4.IsLinkLocalUnicast() && !v4.IsLinkLocalMulticast() && !v4.IsUnspecified() && !v4.IsMulticast()
}

// isUsefulPrivateIPv4 reports whether ip is a usable LAN IPv4 for display:
// RFC1918 private only — excludes loopback, link-local (169.254.0.0/16), public, etc.
func isUsefulPrivateIPv4(ip net.IP) bool {
	if !isUsefulLocalIPv4(ip) {
		return false
	}
	// RFC1918 only (not 169.254, not public)
	return ip.To4().IsPrivate()
}

func ipFromAddr(addr net.Addr) net.IP {
	if addr == nil {
		return nil
	}
	switch a := addr.(type) {
	case *net.TCPAddr:
		return a.IP
	case *net.UDPAddr:
		return a.IP
	default:
		return net.ParseIP(GetIpByAddr(addr.String()))
	}
}

// GetLocalIPs returns IPv4 addresses useful for identifying the npc host.
// Prefers the local IP of conn (path toward nps). Falls back to the outbound
// interface and finally an interface scan.
// The connection path and outbound interface IPs are trusted as-is (only
// loopback/link-local/etc. excluded), so non-standard private ranges such as
// 192.169.0.0/16 are kept; the interface scan keeps RFC1918-only filtering.
func GetLocalIPs(conn net.Conn) string {
	seen := make(map[string]struct{})
	ips := make([]string, 0, 4)

	add := func(ip net.IP, privateOnly bool) {
		if privateOnly {
			if !isUsefulPrivateIPv4(ip) {
				return
			}
		} else if !isUsefulLocalIPv4(ip) {
			return
		}
		s := ip.To4().String()
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		ips = append(ips, s)
	}

	// 1) Local endpoint of the bridge connection — usually the real LAN IP used to reach nps.
	if conn != nil {
		add(ipFromAddr(conn.LocalAddr()), false)
		// One correct path IP is enough for most hosts (avoids docker/vm junk alongside).
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// 2) Outbound interface toward the internet (same idea as GetLocalUdpAddr).
	if tmp, err := net.Dial("udp", "114.114.114.114:53"); err == nil {
		add(ipFromAddr(tmp.LocalAddr()), false)
		_ = tmp.Close()
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// 3) Fallback: all up interfaces with private IPv4.
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok {
					add(ipnet.IP, true)
				}
			}
		}
	}

	if len(ips) > 1 {
		sort.Strings(ips)
	}
	return strings.Join(ips, ",")
}

func IsPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

// GetServerIpByClientIp returns an address the SOCKS5 client can use to reach
// the server's UDP relay (the SOCKS5 BND.ADDR).
//
//   - LAN client: the server's intranet IP.
//   - WAN client: the local IP the client actually connected to, which is
//     guaranteed to be reachable because the TCP control connection succeeded.
//     Only when that address is not public (the server sits behind NAT) do we
//     fall back to querying an external "what is my IP" service.
func GetServerIpByClientIp(clientIp, localIp net.IP) string {
	if !IsPublicIP(clientIp) {
		_, ip := GetIntranetIp()
		return ip
	}
	if IsPublicIP(localIp) {
		return localIp.String()
	}
	return GetExternalIp()
}

func PrintVersion() {
	fmt.Printf("Version: %s\nCore version: %s\nSame core version of client and server can connect each other\n", version.VERSION, version.GetVersion())
}
