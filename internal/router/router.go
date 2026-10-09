package router

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"

	"zecode/internal/config"
	"zecode/internal/handler"
)

// Options 路由初始化所需的依赖
type Options struct {
	Config    *config.Config
	StaticFS  fs.FS // 已剥离前缀的静态资源文件系统（嵌入模式）；磁盘模式可为 nil
	Version   string
	BuildTime string
}

// New 构建并返回配置好的 gin.Engine
func New(opts Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// --- 系统接口 ---
	sysH := handler.NewSystemHandler(opts.Version, opts.BuildTime)
	r.GET("/zing", sysH.Zing)

	// --- API 分组 ---
	// 每类码一个子分组，方便以后横向加 barcode / datamatrix 等
	api := r.Group("/api")
	{
		qrH := handler.NewQRCodeHandler(opts.Config)
		qrcode := api.Group("/qrcode")
		{
			qrcode.GET("/generate", qrH.Generate)
			qrcode.POST("/decode", qrH.Decode)
		}

		// 未来扩展示例：
		// barcode := api.Group("/barcode")
		// {
		//     barH := handler.NewBarcodeHandler(opts.Config)
		//     barcode.GET("/generate", barH.Generate)
		//     barcode.POST("/decode", barH.Decode)
		// }
	}

	// --- 静态资源兜底（必须在所有 API 路由注册之后） ---
	mountStatic(r, opts.Config, opts.StaticFS)

	return r
}

// mountStatic 静态资源兜底。
//
// 优先匹配已注册的 API 路由，未命中的交给 FileServer。
// 这样既不会覆盖 /api/*，又能服务 /index.html、/scanner.html、/lib/* 等。
func mountStatic(r *gin.Engine, cfg *config.Config, embeddedFS fs.FS) {
	var root http.FileSystem
	if cfg.UseEmbeddedStatic && embeddedFS != nil {
		root = http.FS(embeddedFS)
	} else {
		root = http.Dir(cfg.StaticDir)
	}
	fileServer := http.FileServer(root)

	r.NoRoute(func(c *gin.Context) {
		// 静态资源只服务 GET / HEAD
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.String(http.StatusNotFound, "404 Not Found")
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
