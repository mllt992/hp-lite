package hp_android_lib

import (
	"hp-lib/net/cmd"
	"hp-lib/util"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Callback interface {
	SendResult(msg string)
}

var (
	cmdClient  *cmd.CmdClient
	stopReconn chan struct{}
	mu         sync.Mutex
)

func Start(c string, callback Callback) {
	mu.Lock()
	defer mu.Unlock()
	if c == "" {
		callback.SendResult("连接码错误")
		return
	}
	log.Printf("使用连接码模式连接")
	base32 := util.DecodeFromLowerCaseBase32(strings.TrimSpace(c))
	con := strings.Split(base32, ",")
	if len(con) != 2 {
		callback.SendResult("连接码错误")
		return
	}
	server := con[0]
	deviceId := con[1]
	split := strings.Split(server, ":")
	if len(split) != 2 {
		callback.SendResult("连接码错误")
		return
	}
	serverPort, atoiErr := strconv.Atoi(split[1])
	if atoiErr != nil || serverPort <= 0 || serverPort > 65535 {
		callback.SendResult("连接码端口非法")
		return
	}

	// 重新 Start 时先把上一次的循环停掉
	if stopReconn != nil {
		select {
		case <-stopReconn:
		default:
			close(stopReconn)
		}
	}
	stopReconn = make(chan struct{})

	cmdClient = cmd.NewCmdClient(callback.SendResult)
	cmdClient.Connect(split[0], serverPort, deviceId)

	go func() {
		// 指数退避
		const (
			backoffMin = 5 * time.Second
			backoffMax = 5 * time.Minute
		)
		nextDelay := backoffMin
		for {
			timer := time.NewTimer(nextDelay)
			select {
			case <-stopReconn:
				timer.Stop()
				return
			case <-timer.C:
				if cmdClient == nil {
					return
				}
				if cmdClient.Stopped {
					return
				}
				if !cmdClient.GetStatus() {
					cmdClient.Connect(split[0], serverPort, deviceId)
					callback.SendResult("中心服务器重连中")
					nextDelay *= 2
					if nextDelay > backoffMax {
						nextDelay = backoffMax
					}
				} else {
					nextDelay = backoffMin
				}
			}
		}
	}()
}

func Close() bool {
	mu.Lock()
	defer mu.Unlock()
	if stopReconn != nil {
		select {
		case <-stopReconn:
		default:
			close(stopReconn)
		}
	}
	if cmdClient != nil {
		cmdClient.Close()
		return true
	}
	return false
}

func GetStatus() bool {
	mu.Lock()
	defer mu.Unlock()
	if cmdClient != nil {
		return cmdClient.GetStatus()
	}
	return false
}
