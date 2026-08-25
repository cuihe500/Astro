# 数据库规范

> GORM + MariaDB（mysql 驱动），无独立迁移工具。

---

## 连接与迁移

- `internal/repository/db.go` 的 `Init()` 建立连接并持有包级变量 `repository.DB`。
- **迁移 = AutoMigrate**：新模型必须加进 `db.go` 的 `AutoMigrate` 列表，没有 SQL 迁移文件。
- 当前 App 从用户级归属切换到项目级归属。启动时只识别同时含旧 `user_id`/`namespace` 且不含 `project_id` 的旧表：仍有活动 App 时拒绝启动；空表才允许删除并由 AutoMigrate 重建。
- 切换前通过 `make legacy-inventory` 只读盘点测试环境；该目标只执行 SQL `SELECT` 与 `kubectl get`，不会删除数据或 Kubernetes 资源。

## 场景：旧 apps 表切换到项目归属

### 1. Scope / Trigger

- 当 `App` 的归属从旧 `user_id + namespace` 改为必填 `project_id` 时，启动必须先执行窄范围 schema gate。
- 该流程只处理明确的旧 `apps` 表，不迁移应用、不创建默认项目，也不删除用户、OAuth 身份或其他表。
- 在专用测试环境切换 schema 前后，需要盘点 Docker 中的数据库及本机 kind 集群，并在用户确认后删除清单中的空旧 Namespace 时，同样适用本节的命令边界。

### 2. Signatures

- 启动入口：`repository.Init(cfg *config.DatabaseConfig) error`
- gate：`prepareAppSchema(db *gorm.DB) error`
- 旧结构判定：`HasTable(&model.App{})`，且同时满足 `HasColumn(&model.App{}, "user_id")`、`HasColumn(&model.App{}, "namespace")`、`!HasColumn(&model.App{}, "project_id")`
- 重建顺序：`prepareAppSchema(db)` 成功后，再执行 `AutoMigrate(&model.User{}, &model.Project{}, &model.App{}, &model.OAuthIdentity{})`
- 只读盘点：`ASTRO_RUNTIME_ENV=test ASTRO_DATABASE_PORT=<宿主端口> make legacy-inventory`
- 定点删除：`ASTRO_RUNTIME_ENV=test LEGACY_NAMESPACE=astro-user-<数字> make legacy-delete-namespace`
- MariaDB 集成测试：向空的临时数据库提供 `ASTRO_DATABASE_*`，并以 `ASTRO_RUNTIME_ENV=test ASTRO_MARIADB_INTEGRATION=1 make test-integration` 显式执行；检测到既有业务表时立即失败，不复用或清空现有数据库。

### 3. Contracts

- gate 只统计 `apps.deleted_at IS NULL` 的活动记录；活动记录大于零时必须失败关闭，不能修改表结构。
- 明确旧结构且活动记录为零时，删除旧 `apps` 表，随后由 `AutoMigrate` 按当前 `model.App` 重建。
- 非明确旧结构时不猜测、不清理，直接交给常规 `AutoMigrate`；任何检查、删表或迁移错误都向 `Init` 调用方返回。
- 切换前只读盘点入口为 `make legacy-inventory`，要求 `ASTRO_RUNTIME_ENV=test`，并按 `ASTRO_DATABASE_PORT` 唯一定位正在发布该宿主端口的 Docker 容器。数据库查询只在容器内使用其现有 MariaDB/MySQL 环境和客户端执行，不向主机传递或输出凭据。
- `ASTRO_KUBERNETES_KUBECONFIG` 可选；未提供时先沿用 kubectl 默认本地 kubeconfig。默认 context 不可用时，本机必须恰好存在一个 kind 集群，目标通过 `kind get kubeconfig` 将配置直接传给 kubectl，不写临时文件。解析出的 current-context 必须匹配 `kind-*`，后续命令显式固定该 context，歧义或缺失均失败，Kubernetes 资源盘点只允许执行 `kubectl get`。
- 每个精确匹配 `astro-user-<数字>` 且带 `managed-by=astro` 标签的旧 Namespace，都必须继续只读列出其中的 Deployment、Service 与 Pod，并明确标记为空或非空；不得读取 Secret 内容。
- 经用户按盘点清单逐项确认后，使用 `make legacy-delete-namespace LEGACY_NAMESPACE=astro-user-<数字>` 定点删除空旧 Namespace。该入口仅允许 `ASTRO_RUNTIME_ENV=test` 和 `kind-*` context，删除前再次校验名称、`managed-by=astro` 标签及 Deployment/Service/Pod 为空，并等待删除完成后通过 `kubectl get --ignore-not-found` 确认资源不存在。

### 4. Validation & Error Matrix

| 条件 | 结果 |
|---|---|
| `apps` 不存在 | gate 成功，`AutoMigrate` 创建当前表 |
| 明确旧结构且存在活动 App | 启动失败，错误包含活动数量和先清理提示 |
| 明确旧结构且没有活动 App | 删除旧表，再由 `AutoMigrate` 重建 |
| schema 查询或活动数量查询失败 | 启动失败，保留底层错误链 |
| 删除旧表失败 | 启动失败，不继续 `AutoMigrate` |
| 同时存在旧列和 `project_id` 等未知/部分切换结构 | gate 不做破坏性处理，由常规迁移结果决定是否启动 |
| 运行环境不是 `test`、端口无效或端口不能唯一定位运行中的容器 | 盘点失败，不执行数据库或 Kubernetes 操作 |
| kubeconfig/context 不可用或解析结果不是 `kind-*` | 盘点与删除均失败，不切换到其他集群 |
| 删除目标名称不匹配 `astro-user-<数字>`、缺少 `managed-by=astro` 或仍有 Deployment/Service/Pod | 拒绝删除 Namespace |
| 删除命令返回成功但 Namespace 仍可查询 | 删除入口返回失败，不报告完成 |

### 5. Good / Base / Bad Cases

- Good：先用 `make legacy-inventory` 确认旧活动 App 为零，启动后得到仅含项目归属字段的新 `apps` 表。
- Good：数据库凭据只在已唯一定位的容器内使用，Kubernetes 命令固定到已验证的 kind context；用户确认后只删除清单中的单个空旧 Namespace。
- Base：全新数据库没有 `apps` 表且没有旧 Namespace，gate 与清理入口均无破坏性操作。
- Bad：旧表仍有活动 App 时自动补默认项目或直接删表；把数据库密码传回宿主或日志；用通配符、selector 或未固定 context 批量删除 Namespace。

### 6. Tests Required

- MariaDB 集成测试：空数据库执行 `Init` 后断言 `projects`、当前 `apps` 表及外键存在；测试入口检测到既有业务表时必须失败。
- 旧表有活动 App：断言 `prepareAppSchema` 返回错误且旧表和记录均保留。
- 旧表仅有软删除 App：断言 gate 允许重建，且当前 `apps.project_id` 为非空外键。
- 回归测试：列检测必须传 `&model.App{}`，避免以表名字符串调用 `HasColumn` 时 GORM 因缺少解析后的 `stmt.Schema` 崩溃。
- 本机测试环境：盘点后断言目标端口只对应一个运行中容器、SQL 全为 `SELECT`、context 匹配 `kind-*`，并列出每个旧 Namespace 的 Deployment/Service/Pod。
- 删除入口：至少验证非 `test` 环境、非法名称、错误标签和非空 Namespace 都失败；合法空 Namespace 删除后查询不到。

### 7. Wrong vs Correct

```go
// Wrong：HasColumn 的列解析需要模型 schema；只传表名可能触发空指针。
migrator.HasColumn("apps", "user_id")

// Correct：所有表/列检查和删表使用同一个模型值。
appModel := &model.App{}
migrator.HasTable(appModel)
migrator.HasColumn(appModel, "user_id")
migrator.DropTable(appModel)
```

```bash
# Wrong：selector 范围过大，也没有固定已验证的 context。
kubectl delete namespace -l managed-by=astro

# Correct：先盘点并取得确认，再通过受限入口删除一个精确名称。
ASTRO_RUNTIME_ENV=test LEGACY_NAMESPACE=astro-user-<数字> make legacy-delete-namespace
```

## 模型（internal/model/model.go）

- 所有模型嵌入 `BaseModel`（ID/CreatedAt/UpdatedAt/DeletedAt 软删除）。
- 字段用 gorm tag 声明约束，敏感字段 json 隐藏：

```go
type User struct {
    BaseModel
    UUID     string `gorm:"type:char(36);uniqueIndex;not null" json:"uuid"`
    Username string `gorm:"size:64;uniqueIndex;not null" json:"username"`
    Password string `gorm:"size:128;not null" json:"-"`   // json:"-" 不外泄
}
```

- 自动生成字段用 GORM Hook（如 `User.BeforeCreate` 生成 UUID）。
- `Project` 以 `(name, user_id)` 唯一、`namespace` 全局唯一；`App` 必须引用 Project，并以 `(name, project_id)` 唯一。
- Project 和 App 使用限制删除外键；二者业务删除使用硬删除，以便名称可复用。

## Repository 模式（internal/repository/）

- 每个领域一个空结构体 + 构造函数：`AppRepository` / `NewAppRepository()`，方法内直接用包级 `DB`。
- 应用查询必须带项目作用域：`GetByProjectID` / `GetByProjectAndID` / `GetByProjectAndName`。
- 项目删除与应用创建通过 `SELECT ... FOR UPDATE` 锁定同一 Project 行，避免空项目检查与应用插入竞态。
- `AppRepository.CreateInProject` 在事务内插入 App 后调用 `beforeCommit(namespace)`；只有 Kubernetes 创建成功且事务提交成功，App 才能被其他请求查询。回调已开始调用 K8s 后，只要事务失败，service 都调用幂等的 `DeleteApp` 清理可能残留的资源；资源不存在时由 adapter 将 NotFound 视为成功。
- 单字段更新用列更新，不整行 Save：

```go
func (r *AppRepository) UpdateStatus(id uint, status string) error {
    return DB.Model(&model.App{}).Where("id = ?", id).Update("status", status).Error
}
```

- 查询必须参数化占位符：`DB.Where("user_id = ? AND name = ?", userID, name)`。**禁止拼接 SQL 字符串。**

## Not-Found 约定

repository 直接返回 GORM 原始错误；由 **service 层** 用 `errors.Is(err, gorm.ErrRecordNotFound)` 判断并转成业务错误码（见 `internal/service/app.go` 的 `DeleteApp`）。

## 禁止

- 拼接 SQL / `Raw()` 裸 SQL。
- 在 handler 或 service 里直接使用 `repository.DB`。
- 忽略 `.Error` 返回值。
