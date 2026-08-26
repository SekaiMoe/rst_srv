package main

import (
	"flag"
	"log"

	"rstserver/internal/api"

	"github.com/gin-gonic/gin"
)

func main() {
	var (
		addr       = flag.String("addr", ":8443", "监听地址")
		mirror     = flag.String("mirror", "../mirror", "CDN 镜像目录 (含 Android2/ CRI/)")
		dataDir    = flag.String("data", "data", "玩家数据目录")
		masterdata = flag.String("masterdata", "../masterdata.json", "masterdata.json 路径 (空串禁用)")
		debug      = flag.Bool("debug", true, "调试模式日志")
	)
	flag.Parse()

	if !*debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// API 端点 (api.rst-game.com)
	api.RegisterRoutes(r, *dataDir, *masterdata)

	// CDN 静态镜像 (rs.rst-game.com)
	api.RegisterCDN(r, *mirror)

	log.Printf("[rstserver] listening on %s (mirror=%s)", *addr, *mirror)
	if err := r.Run(*addr); err != nil {
		log.Fatal(err)
	}
}
