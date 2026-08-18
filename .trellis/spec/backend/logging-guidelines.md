# 日志规范

> 统一使用 `pkg/logger`（Zap 封装）。禁止标准库 `log` 和 `fmt.Print*` 输出日志。

---

## 用法

`cmd/server/main.go` 启动时 `logger.Init(&cfg.Log)`，退出前 `defer logger.Sync()`。业务代码直接调包级函数：

```go
import "github.com/cuihe500/astro/pkg/logger"

logger.Info("应用创建成功",
    zap.Uint("app_id", app.ID),
    zap.String("name", app.Name),
    zap.String("namespace", app.Namespace))

logger.Error("K8s 创建失败", zap.Error(err), zap.Uint("user_id", userID))
```

## 结构化字段

- 上下文一律用 `zap.Field`（`zap.String` / `zap.Uint` / `zap.Error`），**禁止字符串拼接**（`"user: " + name` ❌）。
- 需要预绑定字段时用 `logger.With(fields...)` 得到子 logger。

## 级别

| 级别 | 用途 |
|---|---|
| Debug | 调试细节，生产默认不输出 |
| Info | 常规业务事件（应用创建、状态同步） |
| Warn | 异常但可继续（状态同步重试、配置降级默认值） |
| Error | 操作失败需要关注（K8s 调用失败、DB 错误） |
| Fatal | 启动阶段致命错误，记录后退出（仅 main.go 初始化用） |

## 输出行为（pkg/logger/logger.go，了解即可）

- 双输出：控制台彩色人类可读 + JSON 文件（配置了 `log.file` 时），文件走 lumberjack 轮转（默认 100MB/10 份/30 天）。
- 级别从 `configs/config.yaml` 的 `log.level` 读取，解析失败降级 Info。

## 禁止

- `log.Println` / `fmt.Println` 打日志。
- 日志中输出密码、Token、JWT Secret 等敏感信息（模型层已用 `json:"-"` 隐藏 Password，日志同样不许打）。
- 在热路径循环里打 Info 及以上级别日志。
