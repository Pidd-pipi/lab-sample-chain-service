# BUG_REPRO

## Bug 现象
服务跑几天内存一直涨；有慢请求时进程关不掉；SHUTDOWN_TIMEOUT_SECONDS 配置不生效。

## 触发方式
- 持续健康检查/访问页面（`recordRequestLog`、`healthProbes` 无上限累积）。
- 有在飞慢请求时触发关停（`shutdownWithTimeout` 用 `context.Background()` 无界等待）。
- 设置 `SHUTDOWN_TIMEOUT_SECONDS=2` 后重启。

## 真实错误信息
- 探针日志与请求日志长度持续增长、无封顶。
- 慢请求在飞时 `Shutdown` 阻塞 30s+，配置的超时完全不生效。
