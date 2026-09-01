package connect

import (
	"bufio"
	"net"
	"runtime"
	"strconv"
	"time"

	net2 "hp-lib/net"
	"hp-lib/protol"

	"github.com/xtaci/smux"
)

type HpTcpConnection struct {
	Enc bool
}

func NewHpTcpConnection() *HpTcpConnection {
	return &HpTcpConnection{}
}

func (connection *HpTcpConnection) ConnectHpTcp(host string, port int, handler net2.HpHandler, call func(mgs string)) *net2.MuxSession {
	// 加 DialTimeout，避免对端 SYN 丢弃时一直阻塞
	conn, err := net.DialTimeout("tcp", host+":"+strconv.Itoa(port), 10*time.Second)
	if err != nil {
		call("不能能连到映射服务器：" + host + ":" + strconv.Itoa(port) + " 原因：" + err.Error())
		return nil
	}

	session, err := smux.Client(conn, nil)
	if err != nil {
		call("不能能连到映射服务器：" + host + ":" + strconv.Itoa(port) + " 原因：" + err.Error())
		_ = conn.Close()
		return nil
	}
	session2 := net2.NewTcpMuxSession(session)
	handler.ChannelActive(session2)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				call("smux accept loop panic: " + string(runtime.Stack(nil, false)))
			}
		}()
		for {
			stream, err := session.AcceptStream()
			if err != nil {
				call(err.Error())
				handler.ChannelInactive(net2.NewTcpMuxStream(stream))
				return
			}
			go func() {
				defer func() {
					if r := recover(); r != nil {
						call("smux stream read loop panic: " + string(runtime.Stack(nil, false)))
					}
				}()
				reader := bufio.NewReader(stream)
				//避坑点：多包问题，需要重复读取解包
				for {
					decode, e := protol.Decode(reader)
					if e != nil {
						handler.ChannelInactive(net2.NewTcpMuxStream(stream))
						return
					}
					if decode != nil {
						handler.ChannelRead(net2.NewTcpMuxStream(stream), decode)
					}
				}
			}()
		}
	}()
	//设置读
	return session2
}
