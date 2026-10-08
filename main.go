package main

import (
	"context"
	"embed"
	"fmt"
	"image"
	"io/fs"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/makiuchi-d/gozxing"
	zxqr "github.com/makiuchi-d/gozxing/qrcode"
	"github.com/skip2/go-qrcode"
)

// ============ 嵌入静态资源 ============
//
// 把整个 statics 目录打进二进制。运行时通过 fs.Sub 去掉 "statics" 前缀，
// 使 URL 路径 /index.html 能正确对应到嵌入文件 statics/index.html。
//
//go:embed statics
var embeddedStatic embed.FS

// ============ 配置 ============

// 通过 -ldflags "-X main.version=... -X main.buildTime=..." 注入
var (
	version   = "dev"
	buildTime = "now"
)

type Config struct {
	Port                 int    `toml:"port"`
	StaticDir            string `toml:"static_dir"`
	UseEmbeddedStatic    bool   `toml:"use_embedded_static"`
	MaxUploadSizeMB      int    `toml:"max_upload_size_mb"`
	DecodeTimeoutSeconds int    `toml:"decode_timeout_seconds"`
	MaxQueryLength       int    `toml:"max_query_length"`
	QRSize               int    `toml:"qr_size"`
}

func defaultConfig() *Config {
	return &Config{
		Port:                 1880,
		StaticDir:            "./statics",
		UseEmbeddedStatic:    true,
		MaxUploadSizeMB:      2,
		DecodeTimeoutSeconds: 5,
		MaxQueryLength:       2048,
		QRSize:               256,
	}
}

const defaultConfigPath = "config/config.toml"

func loadConfig(path string) (*Config, error) {
	cfg := defaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("[警告] 未找到配置文件 %s，将使用内置默认值\n", path)
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}
	return cfg, nil
}

// ============ 静态资源 FileSystem ============

// buildStaticFS 根据配置返回合适的 http.FileSystem：
//   - 嵌入模式：从 embed.FS 中剥离 "statics" 前缀
//   - 磁盘模式：直接读 static_dir 目录
func buildStaticFS(cfg *Config) (http.FileSystem, error) {
	if cfg.UseEmbeddedStatic {
		sub, err := fs.Sub(embeddedStatic, "statics")
		if err != nil {
			return nil, fmt.Errorf("无法挂载嵌入的 statics: %w", err)
		}
		return http.FS(sub), nil
	}
	return http.Dir(cfg.StaticDir), nil
}

// ============ 主入口 ============

func main() {
	configPath := defaultConfigPath
	if v := os.Getenv("ZECODE_CONFIG"); v != "" {
		configPath = v
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	maxUploadSize := int64(cfg.MaxUploadSizeMB) * 1024 * 1024
	decodeTimeout := time.Duration(cfg.DecodeTimeoutSeconds) * time.Second

	// --- 静态文件服务 ---
	staticFS, err := buildStaticFS(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "静态资源初始化失败: %v\n", err)
		os.Exit(1)
	}
	http.Handle("/", http.FileServer(staticFS))

	// --- 1. 生成二维码接口 ---
	http.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.RawQuery) > cfg.MaxQueryLength {
			http.Error(w, "请求过长", http.StatusBadRequest)
			return
		}

		content := r.URL.Query().Get("content")
		if content == "" {
			http.Error(w, "内容不能为空", http.StatusBadRequest)
			return
		}

		pngData, err := qrcode.Encode(content, qrcode.Medium, cfg.QRSize)
		if err != nil {
			http.Error(w, "生成失败", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	})

	// --- 2. 识别二维码接口 ---
	http.HandleFunc("/api/decode", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, fmt.Sprintf("文件过大或格式错误 (最大 %dMB)", cfg.MaxUploadSizeMB), http.StatusRequestEntityTooLarge)
			return
		}

		file, _, err := r.FormFile("qrimage")
		if err != nil {
			http.Error(w, "读取文件失败", http.StatusBadRequest)
			return
		}
		defer file.Close()

		ctx, cancel := context.WithTimeout(context.Background(), decodeTimeout)
		defer cancel()

		type result struct {
			text string
			err  error
		}
		resChan := make(chan result, 1)

		go func() {
			img, _, decodeErr := image.Decode(file)
			if decodeErr != nil {
				resChan <- result{err: decodeErr}
				return
			}

			bmp, bmpErr := gozxing.NewBinaryBitmapFromImage(img)
			if bmpErr != nil {
				resChan <- result{err: bmpErr}
				return
			}

			qrReader := zxqr.NewQRCodeReader()
			resultText, decodeResErr := qrReader.Decode(bmp, nil)
			if decodeResErr != nil {
				resChan <- result{err: decodeResErr}
				return
			}
			resChan <- result{text: resultText.GetText()}
		}()

		select {
		case <-ctx.Done():
			http.Error(w, fmt.Sprintf("识别超时 (超过%d秒)", cfg.DecodeTimeoutSeconds), http.StatusGatewayTimeout)
			return
		case res := <-resChan:
			if res.err != nil {
				http.Error(w, "识别失败: "+res.err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte(res.text))
		}
	})

	// ============ 启动信息 ============
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

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

// ============ 工具函数 ============

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
