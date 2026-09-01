package tunnel

import (
	"hp-server-lib/bean"
	"hp-server-lib/log"
	net2 "hp-server-lib/net/base"
	"hp-server-lib/util"
	"net"
	"strconv"
	"sync"
)

type UdpServer struct {
	cache    sync.Map
	conn     *net2.MuxSession
	udpConn  *net.UDPConn
	userInfo bean.UserConfigInfo
}

func NewUdpServer(conn *net2.MuxSession, userInfo bean.UserConfigInfo) *UdpServer {
	return &UdpServer{
		sync.Map{},
		conn,
		nil,
		userInfo,
	}
}

// ConnectLocal 内网服务的TCP链接
func (udpServer *UdpServer) StartServer(port int) bool {
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Error("不能创建UDP服务器：" + ":" + strconv.Itoa(port) + " 原因：" + err.Error())
		return false
	}
	udpServer.udpConn = conn
	//设置读
	go func() {
		buffer := make([]byte, 8192)
		// 创建缓冲区用于接收数据
		for {
			if udpServer.conn == nil {
				break
			}
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Error("udp读取数据错误 原因：" + err.Error())
				break
			}
			bytes := buffer[:n]
			ip := util.GetClientIPFromUDP(addr)
			// 注意：这里必须用 continue，不能用 break —— break 会跳出整个 read loop，
			// 导致一条被黑白名单命中的包直接干掉整条 UDP 隧道。
			if len(udpServer.userInfo.AllowedIps) > 0 {
				ips := udpServer.userInfo.AllowedIps
				flag := true
				for _, item := range ips {
					if util.IsIPInCIDR(ip, item) {
						flag = false
						break
					}
				}
				if flag {
					continue
				}
			}

			if len(udpServer.userInfo.BlockedIps) > 0 {
				ips := udpServer.userInfo.BlockedIps
				blocked := false
				for _, item := range ips {
					if util.IsIPInCIDR(ip, item) {
						blocked = true
						break
					}
				}
				if blocked {
					continue
				}
			}
			value, ok := udpServer.cache.Load(addr.String())
			if !ok {
				err, handler := NewUdpHandler(udpServer, conn, udpServer.conn, addr, udpServer.userInfo)
				if err != nil {
					log.Error(err.Error())
					break
				}
				handler.ChannelActive(conn)
				udpServer.cache.Store(addr.String(), handler)
				handler.ChannelRead(conn, bytes)
			} else {
				handler := value.(*UdpHandler)
				handler.ChannelRead(conn, bytes)
			}
		}

		udpServer.cache.Range(func(key, value any) bool {
			handler := value.(*UdpHandler)
			handler.ChannelInactive(conn)
			udpServer.cache.Delete(key)
			return true
		})

	}()
	return true
}

func (udpServer *UdpServer) CLose() {
	if udpServer.udpConn != nil {
		udpServer.udpConn.Close()
		udpServer.udpConn = nil
	}
}
