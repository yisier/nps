package goroutine

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/file"
	"github.com/astaxie/beego/logs"
	"github.com/panjf2000/ants/v2"
)

type connGroup struct {
	src    io.ReadWriteCloser
	dst    io.ReadWriteCloser
	wg     *sync.WaitGroup
	n      *int64
	flow   *file.Flow
	task   *file.Tunnel
	remote string
}

//func newConnGroup(dst, src io.ReadWriteCloser, wg *sync.WaitGroup, n *int64) connGroup {
//	return connGroup{
//		src: src,
//		dst: dst,
//		wg:  wg,
//		n:   n,
//	}
//}

func newConnGroup(dst, src io.ReadWriteCloser, wg *sync.WaitGroup, n *int64, flow *file.Flow, task *file.Tunnel, remote string) connGroup {
	return connGroup{
		src:    src,
		dst:    dst,
		wg:     wg,
		n:      n,
		flow:   flow,
		task:   task,
		remote: remote,
	}
}

func CopyBuffer(dst io.Writer, src io.Reader, flow *file.Flow, task *file.Tunnel, remote string) (err error) {
	buf := common.CopyBuff.Get()
	defer common.CopyBuff.Put(buf)
	for {
		if len(buf) <= 0 {
			break
		}
		nr, er := src.Read(buf)

		if task != nil {
			if task.Client.IpWhite && task.Client.IpWhitePass != "" {

				if common.IsAuthIp(remote, task.Client.VerifyKey, task.Client.IpWhiteList) {
					ip := common.GetIpByAddr(remote)
					var jsonBytes []byte

					errorContent, _ := common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "web", "static", "page", "auth.html"))
					authHtml := string(errorContent)
					authHtml = strings.ReplaceAll(authHtml, "${ip}", ip)

					fullRequest := string(buf[0:nr])
					// 获取HTTP请求的第一行
					lines := strings.Split(fullRequest, "\r\n")
					if len(lines) == 0 {
						lines = strings.Split(fullRequest, "\n")
					}
					firstLine := lines[0]

					// 优先处理客户端直接访问的 POST /authIp 请求，直接响应给客户端，不经隧道转发
					if strings.HasPrefix(firstLine, "POST /authIp") {
						pass := ""
						parts := strings.Split(firstLine, " ")
						if len(parts) > 1 {
							path := parts[1]
							if strings.Contains(path, "/authIp?pass=") {
								pass = strings.ReplaceAll(path, "/authIp?pass=", "")
							}
						}
						if pass == task.Client.IpWhitePass {
							task.Client.IpWhiteList = append(task.Client.IpWhiteList, ip)
							file.GetDb().UpdateClient(task.Client)
							logs.Info("客户端IP白名单认证授权成功:vkey [%s] ip [%s] password [%s]", task.Client.VerifyKey, ip, pass)
							jsonBytes, err = json.Marshal(map[string]interface{}{"success": true, "message": "授权成功"})
						} else {
							logs.Error("客户端IP白名单认证授权密码错误:vkey [%s] ip [%s] password [%s]", task.Client.VerifyKey, ip, pass)
							jsonBytes, err = json.Marshal(map[string]interface{}{"success": false, "message": "密码错误"})
						}
						response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(jsonBytes), jsonBytes)
						// 如果 src 是真实的客户端连接（net.Conn），直接写回客户端并关闭连接，避免走隧道转发
						if connSrc, ok := src.(net.Conn); ok {
							connSrc.Write([]byte(response))
							connSrc.Close()
						} else {
							dst.Write([]byte(response))
						}
						return
					}

					// 非授权IP，返回授权页面（同样优先返回给客户端）
					response := fmt.Sprintf("HTTP/1.1 401 Unauthorized\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(authHtml), authHtml)
					if connSrc, ok := src.(net.Conn); ok {
						connSrc.Write([]byte(response))
						connSrc.Close()
					} else {
						dst.Write([]byte(response))
					}
					return
				}
			}
		}

		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				//written += int64(nw)
				if flow != nil {
					flow.Add(int64(nw), int64(nw))
					// <<20 = 1024 * 1024
					if flow.FlowLimit > 0 && (flow.FlowLimit<<20) < (flow.ExportFlow+flow.InletFlow) {
						logs.Error("隧道[%s]流量已经超出", task.Client.VerifyKey)
						break
					}
				}

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
	return err
}

func copyConnGroup(group interface{}) {
	//logs.Info("copyConnGroup.........")
	cg, ok := group.(connGroup)
	if !ok {
		return
	}

	var err error
	err = CopyBuffer(cg.dst, cg.src, cg.flow, cg.task, cg.remote)
	if err != nil {
		cg.src.Close()
		cg.dst.Close()
		//logs.Warn("close npc by copy from nps", err, c.connId)
	}

	//if conns.flow != nil {
	//	conns.flow.Add(in, out)
	//}
	cg.wg.Done()
}

type Conns struct {
	conn1 io.ReadWriteCloser // mux connection
	conn2 net.Conn           // outside connection
	flow  *file.Flow
	wg    *sync.WaitGroup
	task  *file.Tunnel
}

func NewConns(c1 io.ReadWriteCloser, c2 net.Conn, flow *file.Flow, wg *sync.WaitGroup, task *file.Tunnel) Conns {
	return Conns{
		conn1: c1,
		conn2: c2,
		flow:  flow,
		wg:    wg,
		task:  task,
	}
}

func copyConns(group interface{}) {
	//logs.Info("copyConns.........")
	conns := group.(Conns)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	var in, out int64
	remoteAddr := conns.conn2.RemoteAddr().String()
	_ = connCopyPool.Invoke(newConnGroup(conns.conn1, conns.conn2, wg, &in, conns.flow, conns.task, remoteAddr))
	// outside to mux : incoming
	_ = connCopyPool.Invoke(newConnGroup(conns.conn2, conns.conn1, wg, &out, conns.flow, conns.task, remoteAddr))
	// mux to outside : outgoing
	wg.Wait()
	//if conns.flow != nil {
	//	conns.flow.Add(in, out)
	//}
	conns.wg.Done()
}

var connCopyPool, _ = ants.NewPoolWithFunc(200000, copyConnGroup, ants.WithNonblocking(false))
var CopyConnsPool, _ = ants.NewPoolWithFunc(100000, copyConns, ants.WithNonblocking(false))
