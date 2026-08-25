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

### 2. Signatures

- 启动入口：`repository.Init(cfg *config.DatabaseConfig) error`
- gate：`prepareAppSchema(db *gorm.DB) error`
- 旧结构判定：`HasTable(&model.App{})`，且同时满足 `HasColumn(&model.App{}, "user_id")`、`HasColumn(&model.App{}, "namespace")`、`!HasColumn(&model.App{}, "project_id")`
- 重建顺序：`prepareAppSchema(db)` 成功后，再执行 `AutoMigrate(&model.User{}, &model.Project{}, &model.App{}, &model.OAuthIdentity{})`

### 3. Contracts

- gate 只统计 `apps.deleted_at IS NULL` 的活动记录；活动记录大于零时必须失败关闭，不能修改表结构。
- 明确旧结构且活动记录为零时，删除旧 `apps` 表，随后由 `AutoMigrate` 按当前 `model.App` 重建。
- 非明确旧结构时不猜测、不清理，直接交给常规 `AutoMigrate`；任何检查、删表或迁移错误都向 `Init` 调用方返回。
- 切换前只读盘点入口为 `make legacy-inventory`，要求 `ASTRO_RUNTIME_ENV=test`、完整 `ASTRO_DATABASE_*` 连接参数及可读的 `ASTRO_KUBERNETES_KUBECONFIG`。

### 4. Validation & Error Matrix

| 条件 | 结果 |
|---|---|
| `apps` 不存在 | gate 成功，`AutoMigrate` 创建当前表 |
| 明确旧结构且存在活动 App | 启动失败，错误包含活动数量和先清理提示 |
| 明确旧结构且没有活动 App | 删除旧表，再由 `AutoMigrate` 重建 |
| schema 查询或活动数量查询失败 | 启动失败，保留底层错误链 |
| 删除旧表失败 | 启动失败，不继续 `AutoMigrate` |
| 同时存在旧列和 `project_id` 等未知/部分切换结构 | gate 不做破坏性处理，由常规迁移结果决定是否启动 |

### 5. Good / Base / Bad Cases

- Good：先用 `make legacy-inventory` 确认旧活动 App 为零，启动后得到仅含项目归属字段的新 `apps` 表。
- Base：全新数据库没有 `apps` 表，gate 无操作，常规迁移创建 Project 与 App 表。
- Bad：旧表仍有活动 App 时自动补默认项目、静默迁移或直接删表，导致资源失联或数据丢失。

### 6. Tests Required

- MariaDB 集成测试：空数据库执行 `Init` 后断言 `projects`、当前 `apps` 表及外键存在。
- 旧表有活动 App：断言 `prepareAppSchema` 返回错误且旧表和记录均保留。
- 旧表仅有软删除 App：断言 gate 允许重建，且当前 `apps.project_id` 为非空外键。
- 回归测试：列检测必须传 `&model.App{}`，避免以表名字符串调用 `HasColumn` 时 GORM 因缺少解析后的 `stmt.Schema` 崩溃。

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
