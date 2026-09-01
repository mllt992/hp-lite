package cmd

import (
	"bufio"
	net2 "hp-lib/net"
	"hp-lib/protol"
	"net"
	"runtime"
	"strconv"
	"time"
)

type TcpConnection struct {
	conn net.Conn
}

func NewTcpConnection() *TcpConnection {
	t := &TcpConnection{}
	return t
}

func (connection *TcpConnection) Connect(host string, port int, handler net2.Handler, call func(mgs string)) net.Conn {
	// 加 DialTimeout 避免对端 SYN 丢弃时一直阻塞（特别是作为 Windows 服务启动的场景）
	conn, err := net.DialTimeout("tcp", host+":"+strconv.Itoa(port), 10*time.Second)
	if err != nil {
		call("不能能连到服务器：" + host + ":" + strconv.Itoa(port) + " 原因：" + err.Error())
		return nil
	}
	connection.conn = conn
	handler.ChannelActive(conn)
	//设置读
	go func() {
		// 防止 read loop 内任意 panic 把整个客户端静默打死（C-6）
		defer func() {
			if r := recover(); r != nil {
				call("cmd read loop panic: " + string(runtime.Stack(nil, false)))
			}
		}()
		reader := bufio.NewReader(conn)
		for {
			//尝试读检查连接激活
			_, err := reader.Peek(1)
			if err != nil {
				handler.ChannelInactive(conn)
				return
			}
			decode, e := protol.CmdDecode(reader)
			if e != nil {
				call(e.Error())
				handler.ChannelInactive(conn)
				return
			}
			if decode != nil {
				handler.ChannelRead(conn, decode)
			}
		}
	}()
	return conn
}

func (receiver *TcpConnection) Close() {
	if receiver.conn != nil {
		receiver.conn.Close()
	}
}
