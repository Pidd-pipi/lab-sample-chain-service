# BUG_REPRO

## Bug 现象
服务跑久了内存一直涨；导出样品 CSV 或刷新页面越频繁涨得越快；导出内容偶尔串成上一次的；并发导出/刷新有 data race。

## 触发方式
- 反复 `GET /api/samples/export`（`exportLog` 无上限累积，`csvScratch` 共享底层数组）。
- 反复访问 `/` 与 `/app.js`（`visitedPaths` 无上限累积、`staticBuf` 共享）。
- 并发请求导出与静态页。

## 真实错误信息
- `exportLog`/`visitedPaths` 长度持续增长、无封顶。
- 两次导出的响应内容互相串改。
- `go test -race` 并发请求报 `DATA RACE`。
