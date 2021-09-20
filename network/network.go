package network

import (
	"io"
	"net"
	"proxy-mini/global"
)

const (
	KeepAlive        = "KEEP_ALIVE"
	NewConnection    = "NEW_CONNECTION"
	Validation       = "VALIDATION_OK"
	ValidationString = "root;jxlib@2535008"
)

func CreateTCPListener(addr string) (*net.TCPListener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	return tcpListener, nil
}

func CreateTCPConn(addr string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	tcpListener, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	return tcpListener, nil
}

func Join2Conn(local *net.TCPConn, remote *net.TCPConn) {
	global.Logger.Info("[准备转发本地到远程的流量]")
	go joinConn(local, remote)
	global.Logger.Info("[准备转发远程到本地的流量]")
	go joinConn(remote, local)
}

func joinConn(local *net.TCPConn, remote *net.TCPConn) {
	defer local.Close()
	defer remote.Close()
	n, err := io.Copy(local, remote)
	if err != nil {
		global.Logger.Info("[TCP通道拷贝了]", n, "字节")
		global.Logger.Info("[TCP通道连接断开]", err.Error())
		return
	}

	global.Logger.Info("[TCP通道拷贝了]", n, "字节")
}
