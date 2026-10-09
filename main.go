package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"os"

	"zecode/internal/config"
	"zecode/internal/router"
)

// ============ 嵌入静态资源 ============
//
//go:embed statics
var embeddedStatic embed.FS

// 通过 -ldflags "-X main.version=... -X main.buildTime=..." 注入
var (
	version   = "dev"
	buildTime = "now"
)

const defaultConfigPath = "config/config.toml"

func main() {
	configPath := defaultConfigPath
	if v := os.Getenv("ZECODE_CONFIG"); v != "" {
		configPath = v
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	// 构建嵌入模式的子文件系统（剥离 "statics" 前缀）
	var embeddedFS fs.FS
	if cfg.UseEmbeddedStatic {
		sub, err := fs.Sub(embeddedStatic, "statics")
		if err != nil {
			fmt.Fprintf(os.Stderr, "静态资源初始化失败: %v\n", err)
			os.Exit(1)
		}
		embeddedFS = sub
	}

	r := router.New(router.Options{
		Config:    cfg,
		StaticFS:  embeddedFS,
		Version:   version,
		BuildTime: buildTime,
	})

	printStartupInfo(cfg, configPath)

	if err := r.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

// ============ 启动信息 ============

func printStartupInfo(cfg *config.Config, configPath string) {
	fmt.Println("========================================")
	fmt.Println("  ZeCode 已启动")
	fmt.Println("========================================")
	fmt.Printf("版本: %s  (构建于 %s)\n", version, buildTime)
	fmt.Printf("配置文件: %s\n", configPath)
	if cfg.UseEmbeddedStatic {
		fmt.Println("静态资源: 嵌入模式 (已编译进二进制)")
	} else {
		fmt.Printf("静态资源: 磁盘模式 (%s)\n", cfg.StaticDir)
	}
	fmt.Printf("运行参数: 上传上限 %dMB / 识别超时 %ds / 二维码尺寸 %dpx\n",
		cfg.MaxUploadSizeMB, cfg.DecodeTimeoutSeconds, cfg.QRSize)
	fmt.Println("----------------------------------------")

	ips := GetLocalIPv4s()
	if len(ips) == 0 {
		fmt.Printf("本机监听: http://localhost:%d  (未检测到局域网 IP)\n", cfg.Port)
	} else {
		fmt.Println("可通过以下地址访问:")
		for i, ip := range ips {
			fmt.Printf("  [%d] http://%s:%d\n", i+1, ip, cfg.Port)
		}
		fmt.Printf("  [*] http://localhost:%d  (本机调试)\n", cfg.Port)
	}
	fmt.Println("========================================")
}

// GetLocalIPv4s 返回本机所有非环回 IPv4 地址
func GetLocalIPv4s() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if !ipnet.IP.IsLoopback() {
				if ip := ipnet.IP.To4(); ip != nil {
					ips = append(ips, ip.String())
				}
			}
		}
	}
	return ips
}
