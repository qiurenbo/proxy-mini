// 代码部分源于 https://www.jianshu.com/p/ecda849a49bd?%20%E4%BD%9C%E8%80%85%EF%BC%9A%E6%9C%88%E7%90%83%E7%8C%AA%E7%8C%AA%20https://www.bilibili.com/read/cv6213562%20%E5%87%BA%E5%A4%84%EF%BC%9Abilibili
package main

import (
	"bufio"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"proxy-mini/common"
	"proxy-mini/global"
	"proxy-mini/network"
)

/*
* 做图于 http://asciiflow.cn/
* 原理图，1-4 数字代连接建立顺序
*                                1                                   4
        2  8007端口         +-----------+                       +-----------+
User +-----> Server+-----> |  通讯通道   | +-----> Client +--->  |  传输通道  | +---> Server2
               +           +-----------+           ^           +-----------+
               |         8009                      |              10.120.2539.220:80
               |                                   |
               |                 3                 |
               |           +-----------+           |
               +---------> |  传输通道  | +---------+
                           +-----------+
                         8008

User 访问公网服务器 8007 地址
1. 服务器把这个连接放入连接池；
2. 立刻启动传输通道的准备工作（监听端口 8008 ）；
3. 把然后通过通讯通道 8009 告诉客户端，让其连接服务器 8008，建立连接。然后将连接池中的 tcp 连接与客户端的连接进行关联
其中核心是 Join2Conn 调用的 io.Copy 方法，他可以将 tcp 内容体（我想的）发送给目的ip。同样也可以读取目的ip的响应传送
给来源地址
4. 客服端和内网服务器 10.120.2539.220:80 也建立转发上游请求并把内网响应原路返回出去
*/

const (
	controlAddr = "0.0.0.0:8009"
	tunnelAddr  = "0.0.0.0:8008"
	visitAddr   = "0.0.0.0:8007"
)

var (
	clientConn         *net.TCPConn
	connectionPool     map[string]*ConnMatch
	connectionPoolLock sync.Mutex
)

type ConnMatch struct {
	addTime time.Time
	accept  *net.TCPConn
}

func main() {
	global.Logger = common.InitLogger()
	connectionPool = make(map[string]*ConnMatch, 32)
	go createControlChannel()
	go acceptUserRequest()
	go acceptClientRequest()
	cleanConnectionPool()
}

// 创建一个控制通道，用于传递控制消息，如：心跳，创建新连接
func createControlChannel() {
	tcpListener, err := network.CreateTCPListener(controlAddr)
	if err != nil {
		panic(err)
	}

	global.Logger.Info("[已监听]" + controlAddr)
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			global.Logger.Info(err)
			continue
		}
		global.Logger.Info("[新连接]" + tcpConn.RemoteAddr().String())

		// 验证账号密码
		reader := bufio.NewReader(tcpConn)

		s, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			global.Logger.Info("[账号密码读取失败]" + tcpConn.RemoteAddr().String() + s + err.Error())
			return
		}

		// 账号密码验证不通过则返回
		if s != network.ValidationString+"\n" {
			global.Logger.Info("[账号密码验证失败]" + tcpConn.RemoteAddr().String() + s)
			return
		}

		global.Logger.Info("[账号密码验证成功]" + tcpConn.RemoteAddr().String())

		_, err = tcpConn.Write(([]byte)(network.Validation + "\n"))

		if err != nil || err == io.EOF {
			global.Logger.Info("[验证结果传输失败]" + tcpConn.RemoteAddr().String())
			return
		}

		// 如果当前已经有一个客户端存在，则丢弃这个链接
		if clientConn != nil {
			_ = tcpConn.Close()
		} else {
			clientConn = tcpConn
			go keepAlive()
		}
	}
}

// 和客户端保持一个心跳链接
func keepAlive() {
	go func() {
		for {
			if clientConn == nil {
				return
			}
			_, err := clientConn.Write(([]byte)(network.KeepAlive + "\n"))
			if err != nil {
				global.Logger.Info("[已断开客户端连接]", clientConn.RemoteAddr())
				clientConn = nil
				return
			}
			time.Sleep(time.Second * 3)
		}
	}()
}

// 监听来自用户的请求
func acceptUserRequest() {
	tcpListener, err := network.CreateTCPListener(visitAddr)
	if err != nil {
		panic(err)
	}
	defer tcpListener.Close()
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		addConn2Pool(tcpConn)
		sendMessage(network.NewConnection + "\n")
	}
}

// 将用户来的连接放入连接池中
func addConn2Pool(accept *net.TCPConn) {
	connectionPoolLock.Lock()
	defer connectionPoolLock.Unlock()

	now := time.Now()
	connectionPool[strconv.FormatInt(now.UnixNano(), 10)] = &ConnMatch{now, accept}
}

// 发送给客户端新消息
func sendMessage(message string) {
	if clientConn == nil {
		global.Logger.Info("[无已连接的客户端]")
		return
	}
	_, err := clientConn.Write([]byte(message))
	if err != nil {
		global.Logger.Info("[发送消息异常]: message: ", message)
	}
}

// 接收客户端来的请求并建立隧道
func acceptClientRequest() {
	tcpListener, err := network.CreateTCPListener(tunnelAddr)
	if err != nil {
		panic(err)
	}
	defer tcpListener.Close()

	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		go establishTunnel(tcpConn)
	}
}

func establishTunnel(tunnel *net.TCPConn) {
	connectionPoolLock.Lock()
	defer connectionPoolLock.Unlock()

	for key, connMatch := range connectionPool {
		if connMatch.accept != nil {
			go network.Join2Conn(connMatch.accept, tunnel)
			delete(connectionPool, key)
			return
		}
	}

	_ = tunnel.Close()
}

func cleanConnectionPool() {
	for {
		connectionPoolLock.Lock()
		for key, connMatch := range connectionPool {
			if time.Since(connMatch.addTime) > time.Second*10 {
				_ = connMatch.accept.Close()
				delete(connectionPool, key)
			}
		}
		connectionPoolLock.Unlock()
		time.Sleep(5 * time.Second)
	}
}
