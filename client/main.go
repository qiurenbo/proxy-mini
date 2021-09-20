package main

import (
	"bufio"
	"io"
	"net"
	"proxy-mini/common"
	"proxy-mini/global"
	"proxy-mini/network"
)

var (
	// 本地需要暴露的服务端口
	localServerAddr = "127.0.0.1:8000"
	remoteIP        = "localhost"
	// 远端的服务控制通道，用来传递控制信息，如出现新连接和心跳
	remoteControlAddr = remoteIP + ":8009"
	// 远端服务端口，用来建立隧道
	remoteServerAddr = remoteIP + ":8008"
)

func main() {
	global.Logger = common.InitLogger()
	// 连接服务器
	tcpConn, err := network.CreateTCPConn(remoteControlAddr)
	if err != nil {
		global.Logger.Info("[连接失败]" + remoteControlAddr + err.Error())
		return
	}
	global.Logger.Info("[已连接]" + remoteControlAddr)

	if !auth(tcpConn) {
		return
	}

	global.Logger.Info("[循环等待传输通道建立指令]" + remoteControlAddr)
	reader := bufio.NewReader(tcpConn)
	for {
		s, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}

		// 当有新连接信号出现时，新建一个tcp连接
		if s == network.NewConnection+"\n" {
			global.Logger.Info("[传输通道建立完成]" + remoteControlAddr)
			go connectLocalAndRemote()
		}
	}

	// global.Logger.Info("[已断开]" + remoteControlAddr)
}

func auth(tcpConn *net.TCPConn) bool {
	// //  验证账号密码
	_, err := tcpConn.Write(([]byte)(network.ValidationString + "\n"))

	// _, err = writer.WriteString(network.ValidationString + "\n")

	if err != nil || err == io.EOF {
		global.Logger.Info("[账号密码传输失败]" + remoteControlAddr + err.Error())
		return false
	}

	global.Logger.Info("[等待验证结果回传]" + remoteControlAddr)

	reader := bufio.NewReader(tcpConn)

	s, err := reader.ReadString('\n')
	if err != nil || err == io.EOF {
		global.Logger.Info("[读取结果失败]" + remoteControlAddr + err.Error())
		return false
	}

	global.Logger.Info("[读取结果成功]" + s)

	// 账号密码验证不通过则返回
	if s != network.Validation+"\n" {
		global.Logger.Info("[账号密码验证失败]" + remoteControlAddr + err.Error())
		return false
	}

	global.Logger.Info("[账号密码验证成功]" + s)

	return true
}
func connectLocalAndRemote() {
	local := connectLocal()
	remote := connectRemote()

	// 远端验证不成功也直接杀死进程
	// if !auth(remote) {
	// 	panic("[连接服务端验证失败]")
	// }

	if local != nil && remote != nil {
		global.Logger.Info("[端口流量转发]")
		network.Join2Conn(local, remote)
	} else {
		if local != nil {
			_ = local.Close()
		}
		if remote != nil {
			_ = remote.Close()
		}
	}
}

func connectLocal() *net.TCPConn {
	conn, err := network.CreateTCPConn(localServerAddr)
	if err != nil {
		global.Logger.Info("[连接本地服务失败]" + err.Error())
	}
	return conn
}

func connectRemote() *net.TCPConn {
	conn, err := network.CreateTCPConn(remoteServerAddr)
	if err != nil {
		global.Logger.Info("[连接远端服务失败]" + err.Error())
	}

	return conn
}
