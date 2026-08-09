package main

// embed.go — 内嵌前端产物
//
// 构建流程（见 hacks/deploy/api-ui.sh）:
//   1. cd ui && npm run build
//   2. 把 ui/dist/* 复制到 cmd/api-server/web/
//   3. go build ./cmd/api-server
//
// 运行时静态资源优先级：
//   1. 磁盘上的 ui/dist（开发模式，npm run build 后直接生效，无需重新编译）
//   2. 内嵌的 web/（发布产物）

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed all:web
var webFS embed.FS

// staticFS — 前端静态资源
var staticFS fs.FS

func init() {
	// 开发模式：优先使用磁盘上的 ui/dist（相对 hacks/deploy 工作目录）
	if d, err := os.Stat("ui/dist"); err == nil && d.IsDir() {
		staticFS = os.DirFS("ui/dist")
		return
	}
	// 否则使用嵌入的 web/
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	staticFS = sub
}
