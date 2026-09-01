package hp

import (
	"hp-lib/bean"
	hpMessage "hp-lib/message"
	"hp-lib/net"
	"hp-lib/net/connect"
	handler2 "hp-lib/net/handler"
	"hp-lib/protol"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/xtaci/smux"
	"golang.org/x/time/rate"
)

type HpClient struct {
	quit       chan struct{}
	CallMsg    func(message string)
	conn       *net.MuxSession
	quicStream *quic.Stream
	tcpStream  *smux.Stream
	syncLock   sync.Mutex
	Data       *bean.LocalInnerWear
	handler    *handler2.HpClientHandler
}

func NewHpClient(callMsg func(message string)) *HpClient {
	return &HpClient{
		CallMsg: callMsg,
		// 关键：quit 必须初始化，否则 router_table.go 里的 close(oldHpClient.quit) 会 panic。
		quit: make(chan struct{}),
	}
}

func (hpClient *HpClient) Connect(data *bean.LocalInnerWear) {
	if hpClient.conn != nil {
		hpClient.conn.Close()
	}
	hpClient.Data = data

	handler := &handler2.HpClientHandler{
		Key:          data.ConfigKey,
		LocalAddress: data.LocalAddress,
		CallMsg:      hpClient.CallMsg,
	}
	//限速测试
	if data.InLimit > 0 {
		handler.InLimit = rate.NewLimiter(rate.Limit(float64(data.InLimit)), data.InLimit)
	}
	if data.OutLimit > 0 {
		handler.OutLimit = rate.NewLimiter(rate.Limit(float64(data.OutLimit)), data.OutLimit)
	}
	hpClient.handler = handler
	if data.TunType == "TCP" {
		connection := connect.NewHpTcpConnection()
		hpClient.tcpStream = nil
		hpClient.conn = connection.ConnectHpTcp(data.ServerIp, data.ServerPort, handler, hpClient.CallMsg)
	} else {
		connection := connect.NewHpQuicConnection()
		hpClient.quicStream = nil
		hpClient.conn = connection.ConnectHpQuic(data.ServerIp, data.ServerPort, handler, hpClient.CallMsg)
	}
}

func (hpClient *HpClient) GetStatus() bool {
	hpClient.syncLock.Lock()
	defer hpClient.syncLock.Unlock() // 确保锁最终释放
	if hpClient.handler == nil || hpClient.conn == nil {
		return false
	}
	// 每次心跳都开新 stream、写完就关。旧实现复用心跳 stream，
	// 但 server 端 register 流程不回响应、stream 永远不释放，
	// 每次 reconnect 还把它丢给 GC，相当于每个心跳泄漏 1 个 smux stream，跑几天就打满。
	if hpClient.conn.IsTcp() {
		if hpClient.conn.TcpSession == nil {
			return false
		}
		stream, err := hpClient.conn.TcpSession.OpenStream()
		if err != nil {
			hpClient.CallMsg("创建TCP检查流失败:" + err.Error())
			return false
		}
		_, werr := stream.Write(protol.Encode(&hpMessage.HpMessage{Type: hpMessage.HpMessage_KEEPALIVE}))
		_ = stream.Close()
		if werr != nil {
			hpClient.CallMsg("TCP发送心跳包错误:" + werr.Error())
			return false
		}
		return true
	}
	if hpClient.conn.QuicSession == nil {
		return false
	}
	stream, err := hpClient.conn.QuicSession.OpenStream()
	if err != nil {
		hpClient.CallMsg("创建QUIC检查流失败:" + err.Error())
		return false
	}
	_, werr := stream.Write(protol.Encode(&hpMessage.HpMessage{Type: hpMessage.HpMessage_KEEPALIVE}))
	_ = stream.Close()
	if werr != nil {
		hpClient.CallMsg("QUIC发送心跳包错误:" + werr.Error())
		return false
	}
	return true
}

func (hpClient *HpClient) Close() {
	// 关 conn 前先把复用的 stream 句柄清掉，避免遗留
	hpClient.syncLock.Lock()
	hpClient.tcpStream = nil
	hpClient.quicStream = nil
	hpClient.syncLock.Unlock()

	if hpClient.conn != nil {
		hpClient.conn.Close()
		if hpClient.handler != nil {
			hpClient.handler.CloseAll()
		}
		hpClient.conn = nil
	}
}
