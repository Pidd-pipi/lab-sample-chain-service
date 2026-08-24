# Backend

Go 1.22 模块 lab-sample-chain-service，入口为 main.go，启动后提供 /healthz 健康检查。

## Standard Commands

在本目录执行：

    go test ./...
    go build ./...
    go run .

服务端口由 PORT 环境变量控制，默认 8080。静态页面位于 backend/web，由 HTTP 处理器提供。
