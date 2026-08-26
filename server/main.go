package main

import (
	"flag"
	"log"

	"rstserver/internal/api"

	"github.com/gin-gonic/gin"
)

func main() {
	var (
		addr       = flag.String("addr", ":80", "HTTP 监听地址")
		tlsAddr    = flag.String("tlsaddr", ":443", "HTTPS 监听地址 (空串禁用)")
		certFile   = flag.String("cert", "../certs/server.pem", "TLS 证书")
		keyFile    = flag.String("key", "../certs/server.key", "TLS 私钥")
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
	r.Use(func(c *gin.Context) {
		log.Printf("[req] host=%s %s %s ua=%q", c.Request.Host, c.Request.Method, c.Request.URL.Path, c.Request.UserAgent())
		if c.Request.Method == "POST" {
			if err := c.Request.ParseForm(); err == nil {
				log.Printf("[form] %s %v hdrs: UUID=%q TOKEN=%q", c.Request.URL.Path, c.Request.PostForm, c.GetHeader("UUID"), c.GetHeader("TOKEN"))
			}
		}
		c.Next()
	})
	r.Use(gin.Logger(), gin.Recovery())

	// API 端点 (api.rst-game.com)
	api.RegisterRoutes(r, *dataDir, *masterdata)

	// CDN 静态镜像 (rs.rst-game.com)
	api.RegisterCDN(r, *mirror)

	log.Printf("[rstserver] listening on %s (mirror=%s)", *addr, *mirror)
	if *tlsAddr != "" {
		go func() {
			log.Printf("[rstserver] tls listening on %s", *tlsAddr)
			if err := r.RunTLS(*tlsAddr, *certFile, *keyFile); err != nil {
				log.Printf("[tls] %v", err)
			}
		}()
	}
	if err := r.Run(*addr); err != nil {
		log.Fatal(err)
	}
}
