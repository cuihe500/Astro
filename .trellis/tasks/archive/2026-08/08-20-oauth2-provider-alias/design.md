# OAuth2 公开 Provider 别名技术设计

## 1. 变更边界

本任务只调整 OAuth2 的公开标识边界，不改变 OAuth2/OIDC 协议、JWT、用户创建、数据库模型或 Provider 端点配置。

预计修改：

- `internal/service/oauth2.go`：解析公开别名，并区分公开别名与内部 Provider 键。
- `internal/service/oauth2_test.go`：覆盖别名映射、state 和旧公开路径拒绝行为。
- `internal/handler/user.go`：Swagger 示例改为公开别名；现有通用路由形状不变。
- `web/src/features/auth/api.ts`、`web/src/app/router.tsx`：只使用公开别名。
- `configs/config.yaml`、`web/README.md`、`.trellis/spec/backend/auth-guidelines.md`：更新回调契约。
- `docs/`：通过 `make swagger` 同步生成文档。

不修改 `OAuthIdentity` 模型、Repository、迁移和现有用户数据。

## 2. 名称边界

| 层级 | 使用值 | 是否可被客户看到 |
|---|---|---|
| 产品品牌 | `BytCloud Auth` | 是，界面文案 |
| 公开 URL/API 别名 | `bytcloudauth` | 是，URL、网络请求、state |
| 内部 Provider 键 | `authentik` | 否，仅后端配置、映射和数据库 |

公开路由仍使用 `/oauth2/:provider/...`，但 `:provider` 的语义从“配置键”改为“公开别名”。首版只允许 `bytcloudauth`。

## 3. 后端映射

在 OAuth2 service 内增加最小解析函数，将公开别名映射为内部 Provider 键。首版只保留一个显式映射，不新增配置表或抽象层：

```go
const (
    bytCloudAuthAlias       = "bytcloudauth"
    bytCloudAuthProviderKey = "authentik"
)

func resolveOAuth2Provider(alias string) (string, error) {
    if alias != bytCloudAuthAlias {
        return "", errcode.NewWithMsg(errcode.ErrBadRequest, "OAuth2 Provider 不可用")
    }
    return bytCloudAuthProviderKey, nil
}
```

后续确有第二个 Provider 时，再把该函数扩展为 `switch` 或小型映射；当前不为假设需求增加配置结构。

### BuildAuthURL

```text
公开别名 bytcloudauth
  -> resolveOAuth2Provider
  -> 内部键 authentik
  -> 读取 oauth2.providers.authentik
  -> 使用公开别名生成 state
  -> 返回 Provider 授权 URL
```

state 绑定公开别名，可防止不同公开 Provider 回调混用，同时不会泄露内部实现。

### Callback

```text
公开别名 bytcloudauth + code + state
  -> resolveOAuth2Provider
  -> 使用公开别名校验 state
  -> 使用内部键 authentik 读取配置并交换 token
  -> 使用内部键 authentik 查询/创建 OAuthIdentity
  -> 签发现有 Astro JWT
```

解析失败时返回统一 `ErrBadRequest`，错误消息不拼接输入别名或内部键。

## 4. 前端数据流

- 将前端常量改为公开别名，例如 `BYTCLOUD_PROVIDER_ALIAS = "bytcloudauth"`。
- 登录按钮请求 `/api/v1/oauth2/bytcloudauth/login`。
- React callback loader 只接受路由参数 `bytcloudauth`，然后请求 `/api/v1/oauth2/bytcloudauth/callback`。
- 删除前端对内部名称的正则替换；内部名称不再进入前端源码或后端公开错误，自然无需客户端补救。
- 保持通用 `/oauth2/:provider/callback` 路由形状，以便未来增加另一个公开别名，但不实现 Provider 列表或选择器。

## 5. 配置与回调

后端内部配置键保持不变，仅更新 `redirect_url`：

```yaml
oauth2:
  providers:
    authentik:
      redirect_url: "http://localhost:5173/oauth2/bytcloudauth/callback"
```

生产环境 Provider 登记地址记录为：

```text
https://astro.bytcloud.org/oauth2/bytcloudauth/callback
```

OAuth2 `redirect_uri` 必须与 Provider 端登记值完全一致。本任务不新增生产配置文件或提交真实凭据。

## 6. 兼容性与安全

- `/api/v1/oauth2/authentik/...` 不再工作，这是满足隐藏内部名称的有意破坏性变更，不提供兼容重定向。
- 数据库仍以 `authentik + provider_user_id` 命中既有 OAuthIdentity，因此无需迁移且不会创建重复用户。
- state 的 HMAC、过期时间和 nonce 逻辑保持不变，只把绑定值改为公开别名。
- authorization code、state、access token、client secret 和 Astro JWT 继续不得进入日志或页面。

## 7. 验证与回滚

验证：

- Go 单元测试确认别名解析、旧内部键拒绝、state 绑定公开别名。
- 前端测试/构建确认认证 API 路径和类型无回归。
- Swagger 生成后确认公开示例使用 `bytcloudauth`。
- 浏览器 Mock API 验证登录入口、成功回调和错误回调 URL。
- 检查前端源码及构建产物不包含内部名称。

回滚时恢复前端公开常量、配置回调地址和 service 直接使用 Provider 参数的行为即可；没有数据库回滚。
