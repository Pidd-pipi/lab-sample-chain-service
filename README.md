# Lab Sample Chain Service

实验室样品链路 HTTP 服务，提供样品集合查询、状态推进、健康检查和静态操作页面。

## Quick Start

从项目根目录执行：

    cp .env.example .env
    cd backend
    go test ./...
    go build ./...
    go run .

默认监听 PORT 指定的端口，未设置时为 8080。健康检查：GET /healthz。

## API

GET /api/samples；POST /api/samples/{id}/status（状态：in_review、released、rejected）。根路径提供静态页面，错误响应保持现有接口约定。

## Layout

    .
    ├── backend/               Go 模块、源码、静态资源和 Dockerfile
    │   ├── go.mod
    │   ├── *.go
    │   ├── web/               页面资源
    │   └── Dockerfile
    ├── database/README.md     持久化说明
    ├── output/verification.md 验证记录
    ├── prompt.txt             原始任务题面
    └── runtime_smoke.json     启动契约

静态页面位于 backend/web，由 HTTP 处理器提供。

## Verification

验证命令和真实启动结果记录在 output/verification.md。
