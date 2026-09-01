package cmd

import (
	net2 "hp-lib/net/hp"
)

var version = "hp-lite:6.0"

type CmdClient struct {
	CallMsg    func(message string)
	handler    *CmdClientHandler
	Connection *TcpConnection
	// Stopped 由 server DISCONNECT 消息置 true；reconnect 循环会查这个标志，置 true 后不再重连
	Stopped bool
}

func NewCmdClient(callMsg func(message string)) *CmdClient {
	return &CmdClient{
		CallMsg: callMsg,
	}
}

// Connect 拨号到 server。
// 关键：先 Dial 成功再替换 handler / Connection，否则 Dial 失败时旧 handler.Ide() 会被新 handler 覆盖，
// 而旧 handler 的 Conn 可能已经被关掉，下一次心跳调用会 nil pointer panic。
func (cmdClient *CmdClient) Connect(serverIp string, serverPort int, key string) {
	if cmdClient.Connection != nil {
		cmdClient.Connection.Close()
		cmdClient.Connection = nil
	}
	connection := NewTcpConnection()
	handler := &CmdClientHandler{
		Key:       key,
		CmdClient: cmdClient,
	}
	// 拨号；如果失败保持旧的 handler/Connection 不动
	conn := connection.Connect(serverIp, serverPort, handler, cmdClient.CallMsg)
	if conn == nil {
		// 拨号失败，handler 的 Active 仍是 false。下次 GetStatus 会返回 false，重连循环会再来一次
		return
	}
	// 拨号成功，再覆盖全局状态
	cmdClient.handler = handler
	cmdClient.Connection = connection
}

func (cmdClient *CmdClient) GetStatus() bool {
	if cmdClient.handler == nil {
		return false
	}
	return cmdClient.handler.Ide()
}

func (cmdClient *CmdClient) Close() {
	net2.CloseTunnel()
	if cmdClient.Connection != nil {
		cmdClient.Connection.Close()
		cmdClient.Connection = nil
	}
}
