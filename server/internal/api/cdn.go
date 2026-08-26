package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// RegisterCDN — 挂载 rs.rst-game.com 静态镜像
//
// 镜像结构 (mirror/):
//
//	Android2/<name>   AssetBundle + assetbundle_versions.txt
//	CRI/<name>        .acb/.awb/.usm 音频视频
//	index.html        原站根页面
func RegisterCDN(r *gin.Engine, mirrorDir string) {
	// Range 请求支持由 http.FileServer 原生处理 (UnityWebRequest 断点续传)
	for _, sub := range []string{"Android2", "CRI"} {
		dir := filepath.Join(mirrorDir, sub)
		if _, err := os.Stat(dir); err != nil {
			continue // 目录不存在则跳过 (允许只跑 API)
		}
		r.Static(sub, dir)
	}
	// 根页面
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(mirrorDir, "index.html"))
	})
	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
}
