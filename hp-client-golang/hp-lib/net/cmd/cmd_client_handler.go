package cmd

import (
	"hp-lib/bean"
	cmdMessage "hp-lib/message"
	"hp-lib/net/hp"
	"hp-lib/protol"
	"hp-lib/util"
	"net"
	"os"
	"time"
)

type CmdClientHandler struct {
	Key       string
	CmdClient *CmdClient
	Conn      net.Conn
	Active    bool
}

// ChannelActive 连接激活时，发送注册信息到云端
func (h *CmdClientHandler) ChannelActive(conn net.Conn) {
	h.Conn = conn
	h.Active = true
	message := &cmdMessage.CmdMessage{
		Version: version,
		Type:    cmdMessage.CmdMessage_CONNECT,
		Key:     h.Key,
		Data:    util.SysInfo(),
	}
	_, err := conn.Write(protol.CmdEncode(message))
	if err != nil {
		h.Active = false
		return
	}
	// 刚连上时不要立刻做一次心跳，否则 server 端日志会被噪音淹没。
}

// Ide 主动探测连接是否存活，返回 true 表示可用。
// 旧实现每 10s 发一次 TIPS，把系统信息当心跳 payload；改成"失败即重连"语义，不再主动刷消息。
func (h *CmdClientHandler) Ide() bool {
	if !h.Active {
		return false
	}
	if h.Conn == nil {
		// 之前 Close 把 Conn 置 nil 但忘了把 Active 置 false —— 修一下，避免后面 nil Write panic
		h.Active = false
		return false
	}
	// 设置一个非常短的超时；写不进就说明链路已经半死
	_ = h.Conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_, err := h.Conn.Write(protol.CmdEncode(&cmdMessage.CmdMessage{Version: version, Key: h.Key, Type: cmdMessage.CmdMessage_TIPS, Data: util.SysInfo()}))
	_ = h.Conn.SetWriteDeadline(time.Time{})
	if err != nil {
		h.CmdClient.CallMsg("中心节点心跳异常:" + err.Error())
		h.Active = false
		// 关掉 conn，让 reconnect 循环能感知到
		_ = h.Conn.Close()
		return false
	}
	return true
}

func (h *CmdClientHandler) ChannelRead(conn net.Conn, data interface{}) {
	message := data.(*cmdMessage.CmdMessage)
	switch message.Type {
	case cmdMessage.CmdMessage_DISCONNECT:
		// 服务器要求客户端直接停掉进程。按业务需求保留 os.Exit 语义。
		h.CmdClient.CallMsg("服务器要求你关闭：" + message.GetData())
		os.Exit(-1)
	case cmdMessage.CmdMessage_TIPS:
		h.CmdClient.CallMsg(message.Data)
		return
	case cmdMessage.CmdMessage_LOCAL_INNER_WEAR:
		h.CmdClient.CallMsg("正在检查本地映射配置关系")
		h.connected(message)
		return
	default:
		h.CmdClient.CallMsg("未知类型数据：" + message.GetData())
	}
}

func (h *CmdClientHandler) ChannelInactive(conn net.Conn) {
	h.Active = false
	if conn != nil {
		_ = conn.Close()
	}
}

func (h *CmdClientHandler) connected(message *cmdMessage.CmdMessage) {
	wear := bean.NewLocalInnerWear(message.Data)
	hp.RefreshRouter(wear, h.CmdClient.CallMsg)
}
