package main

import (
	"flag"
	"fmt"
	"hp-lib/net/cmd"
	"hp-lib/util"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kardianos/service"
)

// win-> eventvwr.msc
var logger service.Logger

// 定义服务配置
type program struct {
	serverIp   string
	serverPort int
	deviceId   string
	cmdClient  *cmd.CmdClient
	stopChan   chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

// 实现 service.Interface 接口的 Start 方法
func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("服务以交互模式启动")
	} else {
		logger.Info("服务启动成功")
	}
	go p.run()
	return nil
}

// 实现 service.Interface 接口的 Stop 方法
func (p *program) Stop(s service.Service) error {
	if service.Interactive() {
		logger.Info("服务以交互模式停止")
	} else {
		logger.Info("服务正在停止")
	}
	// 用 sync.Once 避免 kardianos 在异常情况下重复调用 Stop 导致 close panic
	p.stopOnce.Do(func() {
		close(p.stopChan)
	})
	// 等 run() 完全退出再返回，确保 reconnect goroutine 不会在进程退出后还在 Dial
	p.wg.Wait()
	return nil
}

// 服务核心运行逻辑
func (p *program) run() {
	defer p.wg.Done()
	p.cmdClient = cmd.NewCmdClient(func(message string) {
		logger.Info(message)
	})

	p.cmdClient.Connect(p.serverIp, p.serverPort, p.deviceId)
	logger.Infof("已连接到服务器 %s:%d (设备ID: %s)", p.serverIp, p.serverPort, p.deviceId)

	// 重连循环
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// 指数退避：连续失败时 sleep 拉长，最长 5 分钟
		const (
			backoffMin = 5 * time.Second
			backoffMax = 5 * time.Minute
		)
		nextDelay := backoffMin
		sleepUntil := time.Time{}

		for {
			select {
			case <-p.stopChan:
				logger.Info("重连循环已停止")
				return
			case <-ticker.C:
				// 被 server 主动要求下线就停
				if p.cmdClient != nil && p.cmdClient.Stopped {
					logger.Info("收到 server DISCONNECT，重连循环退出")
					return
				}
				// 还在退避期就跳过
				if !sleepUntil.IsZero() && time.Now().Before(sleepUntil) {
					continue
				}
				if p.cmdClient != nil && !p.cmdClient.GetStatus() {
					logger.Info("与服务器断开连接，正在重连...")
					p.cmdClient.Connect(p.serverIp, p.serverPort, p.deviceId)
					sleepUntil = time.Now().Add(nextDelay)
					nextDelay *= 2
					if nextDelay > backoffMax {
						nextDelay = backoffMax
					}
				} else {
					// 连上了就重置退避
					nextDelay = backoffMin
					sleepUntil = time.Time{}
				}
			}
		}
	}()

	<-p.stopChan
	logger.Info("服务核心逻辑已停止")
}

// 解析连接码并返回服务器信息
func parseConnectionCode(c string) (serverIp string, serverPort int, deviceId string, err error) {
	if c == "" {
		return "", 0, "", fmt.Errorf("连接码为空")
	}

	base32 := util.DecodeFromLowerCaseBase32(strings.TrimSpace(c))
	conn := strings.Split(base32, ",")

	if len(conn) != 2 {
		return "", 0, "", fmt.Errorf("连接码格式错误：分割后长度不为2（实际：%d）", len(conn))
	}

	server := conn[0]
	deviceId = conn[1]

	split := strings.Split(server, ":")
	if len(split) != 2 {
		return "", 0, "", fmt.Errorf("服务器地址格式错误：%s（应为 IP:端口）", server)
	}

	serverIp = split[0]
	port, err := strconv.Atoi(split[1])
	if err != nil {
		return "", 0, "", fmt.Errorf("端口号不是有效数字：%s，错误：%v", split[1], err)
	}

	if port < 1 || port > 65535 {
		return "", 0, "", fmt.Errorf("端口号无效：%d（必须在 1-65535 之间）", port)
	}

	return serverIp, port, deviceId, nil
}

func main() {
	// ########## 关键修复：只调用一次 flag.Parse() ##########
	var (
		c             string // 连接码
		serviceAction string // 服务操作
	)

	// 1. 定义所有命令行参数（一次定义完成）
	flag.StringVar(&c, "c", "", "连接码（必填，安装服务时固化）")
	flag.StringVar(&serviceAction, "action", "", "服务操作：install/start/stop/uninstall/status")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "使用方法：%s [参数]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "参数说明：")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\n示例：")
		fmt.Fprintln(os.Stderr, "  交互模式运行：", os.Args[0], "-c \"你的连接码\"")
		fmt.Fprintln(os.Stderr, "  安装服务：", os.Args[0], "-c \"你的连接码\" -action install")
		fmt.Fprintln(os.Stderr, "  启动服务：", os.Args[0], "-action start")
		fmt.Fprintln(os.Stderr, "  停止服务：", os.Args[0], "-action stop")
		fmt.Fprintln(os.Stderr, "  查看状态：", os.Args[0], "-action status")
		fmt.Fprintln(os.Stderr, "  卸载服务：", os.Args[0], "-action uninstall")
	}

	// 2. 只解析一次！（这是解决问题的核心）
	flag.Parse()

	// ########## 处理连接码（分场景：服务操作 vs 正常运行）##########
	var (
		serverIp   string
		serverPort int
		deviceId   string
		err        error
	)

	// 场景A：执行服务操作（install/start/stop/uninstall/status）
	if serviceAction != "" {
		switch serviceAction {
		case "start", "stop", "status", "uninstall":
			// 这些操作不需要 -c 参数（start 会使用安装时固化的参数）
			// 但 status/uninstall 等操作需要验证服务是否存在，所以直接创建服务实例即可
			break

		case "install":
			// 安装服务必须指定 -c 参数
			if c == "" {
				c = os.Getenv("c") // 从环境变量 fallback
				if c == "" {
					log.Fatal("错误：安装服务必须通过 -c 参数或 c 环境变量指定连接码")
				}
			}
			// 解析连接码（安装时验证连接码有效性）
			serverIp, serverPort, deviceId, err = parseConnectionCode(c)
			if err != nil {
				log.Fatalf("连接码解析失败：%v", err)
			}

		default:
			log.Fatalf("无效的操作：%s，支持的操作：install/start/stop/uninstall/status", serviceAction)
		}

		// 场景B：无服务操作 → 交互模式运行（必须有连接码）
	} else {
		if c == "" {
			c = os.Getenv("c")
			if c == "" {
				log.Fatal("错误：必须通过 -c 参数或 c 环境变量指定连接码")
			}
		}
		// 解析连接码
		serverIp, serverPort, deviceId, err = parseConnectionCode(c)
		if err != nil {
			log.Fatalf("连接码解析失败：%v", err)
		}
	}

	// ########## 服务配置（固化连接码只在 install 时生效）##########
	serviceConfig := &service.Config{
		Name:        "hp-lite", // 服务唯一标识（不要修改）
		DisplayName: "hp-lite", // 服务显示名称
		Description: "hp-lite 命令行客户端服务，用于与中心服务器通信",
		Arguments:   []string{"-c", c}, // 安装时固化 -c 参数，启动时自动传递
	}
	// 如果是安装操作，更新服务描述（包含连接码，便于排查）
	if serviceAction == "install" {
		serviceConfig.Description += "（连接码：" + c + "）"
	}

	// ########## 创建服务实例 ##########
	// 注意：start/stop/status/uninstall 时，serverIp 等可能为空，但不影响服务操作
	prg := &program{
		serverIp:   serverIp,
		serverPort: serverPort,
		deviceId:   deviceId,
		stopChan:   make(chan struct{}),
	}

	s, err := service.New(prg, serviceConfig)
	if err != nil {
		log.Fatalf("服务创建失败：%v", err)
	}

	// 初始化日志
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatalf("日志初始化失败：%v", err)
	}

	// ########## 执行服务操作 ##########
	switch serviceAction {
	case "install":
		err = s.Install()
		if err != nil {
			log.Fatalf("服务安装失败：%v", err)
		}
		log.Printf("✅ 服务安装成功！服务名称：%s", serviceConfig.Name)
		log.Printf("📌 连接码：%s", c)
		log.Println("💡 后续操作：")
		log.Println("   启动服务：", os.Args[0], "-action start")
		log.Println("   停止服务：", os.Args[0], "-action stop")
		log.Println("   查看状态：", os.Args[0], "-action status")
		log.Println("   卸载服务：", os.Args[0], "-action uninstall")
		log.Println("   [注意事项]：")
		log.Println("   1、安装服务后请不要删除当前文件，如果需要删除当前文件，请先停止服务、然后在卸载服务、最后在删除文件")
		log.Println("   2、更新程序前请先停止服务再替换程序然后再启动服务")

	case "start":
		err = s.Start()
		if err != nil {
			log.Fatalf("服务启动失败：%v", err)
		}
		log.Println("✅ 服务启动成功！可通过 -action status 查看状态")

	case "stop":
		err = s.Stop()
		if err != nil {
			log.Fatalf("服务停止失败：%v", err)
		}
		log.Println("✅ 服务停止成功！")

	case "uninstall":
		err = s.Uninstall()
		if err != nil {
			log.Fatalf("服务卸载失败：%v", err)
		}
		log.Println("✅ 服务卸载成功！")

	case "status":
		status, err := s.Status()
		if err != nil {
			log.Fatalf("获取服务状态失败：%v", err)
		}
		switch status {
		case service.StatusRunning:
			log.Println("🟢 服务状态：运行中")
		case service.StatusStopped:
			log.Println("🔴 服务状态：已停止")
		default:
			log.Printf("🟡 服务状态：%v", status)
		}

	case "":
		// 无操作 → 交互模式运行
		log.Printf("🚀 以交互模式启动（服务器：%s:%d，设备ID：%s）", serverIp, serverPort, deviceId)
		err = s.Run()
		if err != nil {
			log.Fatalf("交互模式运行失败：%v", err)
		}
	}
}
