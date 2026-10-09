package handler

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makiuchi-d/gozxing"
	zxqr "github.com/makiuchi-d/gozxing/qrcode"
	"github.com/skip2/go-qrcode"

	"zecode/internal/config"
)

// QRCodeHandler 封装二维码相关接口
type QRCodeHandler struct {
	cfg *config.Config
}

func NewQRCodeHandler(cfg *config.Config) *QRCodeHandler {
	return &QRCodeHandler{cfg: cfg}
}

// Generate 生成二维码
//
//	GET /api/qrcode/generate?content=xxx
func (h *QRCodeHandler) Generate(c *gin.Context) {
	if len(c.Request.URL.RawQuery) > h.cfg.MaxQueryLength {
		c.String(http.StatusBadRequest, "请求过长")
		return
	}

	content := c.Query("content")
	if content == "" {
		c.String(http.StatusBadRequest, "内容不能为空")
		return
	}

	pngData, err := qrcode.Encode(content, qrcode.Medium, h.cfg.QRSize)
	if err != nil {
		c.String(http.StatusInternalServerError, "生成失败")
		return
	}

	c.Data(http.StatusOK, "image/png", pngData)
}

// Decode 识别二维码
//
//	POST /api/qrcode/decode
//	form-data: qrimage
func (h *QRCodeHandler) Decode(c *gin.Context) {
	maxUploadSize := h.cfg.MaxUploadSize()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		c.String(http.StatusRequestEntityTooLarge,
			fmt.Sprintf("文件过大或格式错误 (最大 %dMB)", h.cfg.MaxUploadSizeMB))
		return
	}

	file, _, err := c.Request.FormFile("qrimage")
	if err != nil {
		c.String(http.StatusBadRequest, "读取文件失败")
		return
	}
	defer file.Close()

	decodeTimeout := time.Duration(h.cfg.DecodeTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(c.Request.Context(), decodeTimeout)
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
		c.String(http.StatusGatewayTimeout,
			fmt.Sprintf("识别超时 (超过%d秒)", h.cfg.DecodeTimeoutSeconds))
	case res := <-resChan:
		if res.err != nil {
			c.String(http.StatusInternalServerError, "识别失败: "+res.err.Error())
			return
		}
		c.String(http.StatusOK, res.text)
	}
}
