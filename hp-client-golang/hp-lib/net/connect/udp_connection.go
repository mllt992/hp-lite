package connect

import (
	"bufio"
	net2 "hp-lib/net"
	"hp-lib/util"
	"io"
	"net"
	"runtime"
	"strconv"
	"time"
)

type UdpConnection struct {
}

func NewUdpConnection() *UdpConnection {
	return &UdpConnection{}
}

func (connection *UdpConnection) Connect(address string, handler net2.Handler, call func(mgs string)) net.Conn {
	err2, _, host, port := util.ProtocolInfo(address)
	if err2 != nil {
		call("地址解析错误：" + host + ":" + strconv.Itoa(port) + " 原因：" + err2.Error())
		return nil
	}

	// UDP 是无连接的，Dial 永远成功但对端可能根本不存在。给 Read 套个超时，
	// 至少能在 N 秒内识别"对端没回"的情况，让上层决定要不要重连。
	conn, err := net.Dial("udp", host+":"+strconv.Itoa(port))
	if err != nil {
		call("不能能连到服务器：" + host + ":" + strconv.Itoa(port) + " 原因：" + err.Error())
		return nil
	}
	handler.ChannelActive(conn)
	//设置读
	go func() {
		defer func() {
			if r := recover(); r != nil {
				call("local udp read loop panic: " + string(runtime.Stack(nil, false)))
			}
		}()
		reader := bufio.NewReader(conn)
		for {
			// 读超时：30 秒没数据就认为这条 UDP 流已经死了
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			_, err := reader.Peek(1)
			if err != nil {
				handler.ChannelInactive(conn)
				return
			}
			_ = conn.SetReadDeadline(time.Time{})
			if reader.Buffered() > 0 {
				data := make([]byte, reader.Buffered())
				if _, rerr := io.ReadFull(reader, data); rerr != nil {
					handler.ChannelInactive(conn)
					return
				}
				handler.ChannelRead(conn, data)
			}
		}
	}()
	return conn
}
