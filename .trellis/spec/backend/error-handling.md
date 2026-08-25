# 错误处理

> 错误码枚举 + 分层转换：repository 返原始错误 → service 包成 errcode → handler 统一输出。

---

## 错误码（pkg/errcode/code.go）

分段规则，新增错误码必须落在对应段并在 `codeMessages` 补默认消息：

| 段 | 含义 | 示例 |
|---|---|---|
| `0` | 成功 | `Success` |
| `1xxxx` | 客户端错误 | `ErrBadRequest=10001`、`ErrUnauthorized=10002` |
| `20xxx` | 用户业务错误 | `ErrUserExists=20001`、`ErrTokenExpired=20011` |
| `21xxx` | 应用业务错误 | `ErrAppNotFound=21001`、`ErrAppCreateFail=21003` |
| `22xxx` | 项目业务错误 | `ErrProjectNotFound=22001`、`ErrProjectNotEmpty=22003` |
| `3xxxx` | 系统错误 | `ErrDatabase=30002`、`ErrK8sOperation=30005` |

核心类型：`errcode.Error{Code, Msg}`，构造用 `errcode.New(code)` / `errcode.NewWithMsg(code, msg)`，`errcode.FromError(err)` 把任意 error 提取为 `*Error`（非 Error 类型归为 `ErrInternal`）。

## 分层职责

**service 层**负责把底层错误翻译成 `*errcode.Error`（`internal/service/app.go`）：

```go
app, err := s.repo.GetByID(appID)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return errcode.New(errcode.ErrAppNotFound)
    }
    return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
}
```

**handler 层**只做两件事：参数错误直接 `BadRequest(c, msg)`；service 错误一律 `HandleError(c, err)`：

```go
if err := c.ShouldBindJSON(&req); err != nil {
    BadRequest(c, "参数错误: "+err.Error())
    return
}
app, err := h.svc.CreateApp(...)
if err != nil {
    HandleError(c, err)   // 自动 FromError 并输出 code/message
    return
}
Success(c, app)
```

## 响应格式（internal/handler/response.go）

所有 API（含错误）返回 **HTTP 200**，业务状态在 body：`{"code": 21001, "message": "应用不存在", "data": ...}`。辅助函数：`Success` / `Error` / `ErrorWithCode` / `BadRequest` / `Unauthorized` / `Forbidden` / `NotFound` / `InternalError` / `HandleError`。中间件中报错后必须 `c.Abort()`（见 `internal/middleware/auth.go`）。

## 规则

- 禁止用 `_` 忽略错误；补偿清理失败也必须显式返回或记录。
- 外部资源已经可能创建后，补偿清理必须使用 `context.WithoutCancel(ctx)`，不能因原请求已取消而跳过；普通查询和用户主动删除仍使用原请求 context。
- 禁止硬编码数字错误码，必须用 `errcode` 枚举。
- 判断特定错误用 `errors.Is`，不用字符串比较。
