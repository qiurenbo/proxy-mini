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

	//  验证账号密码
	_, err = tcpConn.Write(([]byte)(network.ValidationString + "\n"))

	// _, err = writer.WriteString(network.ValidationString + "\n")

	if err != nil || err == io.EOF {
		global.Logger.Info("[账号密码传输失败]" + remoteControlAddr + err.Error())
		return
	}

	global.Logger.Info("[等待验证结果回传]" + remoteControlAddr)

	reader := bufio.NewReader(tcpConn)

	s, err := reader.ReadString('\n')
	if err != nil || err == io.EOF {
		return
	}

	// 账号密码验证不通过则返回
	if s != network.Validation+"\n" {
		global.Logger.Info("[账号密码验证失败]" + remoteControlAddr + err.Error())
		return
	}

	global.Logger.Info("[账号密码验证成功]" + remoteControlAddr)

	// 创建一个 buffer reader 不用手动分配 buffer 了
	// reader := bufio.NewReader(tcpConn)
	global.Logger.Info("[循环等待传输通道建立指令]" + remoteControlAddr)
	for {
		s, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}

		// 当有新连接信号出现时，新建一个tcp连接
		if s == network.NewConnection+"\n" {
			go connectLocalAndRemote()
		}
	}

	global.Logger.Info("[已断开]" + remoteControlAddr)
}

func connectLocalAndRemote() {
	local := connectLocal()
	remote := connectRemote()

	if local != nil && remote != nil {
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
