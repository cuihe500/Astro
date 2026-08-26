//go:build integration

package repository

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type legacyApp struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"not null"`
	Namespace string         `gorm:"size:63;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 将旧模型绑定到生产使用的 apps 表。
func (legacyApp) TableName() string { return "apps" }

func TestMariaDBProjectOwnership(t *testing.T) {
	databaseConfig := integrationDatabaseConfig(t)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		databaseConfig.User, databaseConfig.Password, databaseConfig.Host, databaseConfig.Port,
		databaseConfig.DBName, databaseConfig.Charset)
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("连接 MariaDB 失败: %v", err)
	}
	closeDatabase(t, database)
	for _, table := range []any{&model.App{}, &model.Project{}, &model.OAuthIdentity{}, &model.User{}} {
		if database.Migrator().HasTable(table) {
			t.Fatal("MariaDB 集成测试只允许使用空的临时数据库")
		}
	}

	if err := database.AutoMigrate(&legacyApp{}); err != nil {
		t.Fatalf("创建旧 apps 表失败: %v", err)
	}
	legacy := legacyApp{UserID: 1, Namespace: "astro-user-1"}
	if err := database.Create(&legacy).Error; err != nil {
		t.Fatalf("创建旧应用记录失败: %v", err)
	}
	if err := prepareAppSchema(database); err == nil {
		t.Fatal("存在活动旧应用时 schema gate 应拒绝迁移")
	}
	if !database.Migrator().HasTable(&legacyApp{}) {
		t.Fatal("schema gate 拒绝迁移后不应删除旧 apps 表")
	}
	if err := database.Delete(&legacy).Error; err != nil {
		t.Fatalf("软删除旧应用失败: %v", err)
	}
	if err := prepareAppSchema(database); err != nil {
		t.Fatalf("仅有软删除旧应用时 schema gate 失败: %v", err)
	}
	if database.Migrator().HasTable(&legacyApp{}) {
		t.Fatal("schema gate 未删除已清空的旧 apps 表")
	}

	if err := Init(databaseConfig); err != nil {
		t.Fatalf("初始化当前 schema 失败: %v", err)
	}
	closeDatabase(t, DB)
	DB = DB.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	assertProjectOwnershipSchema(t, DB)
	assertProjectOwnershipConstraints(t)
	assertProjectLockSerializesCreate(t)
}

func integrationDatabaseConfig(t *testing.T) *config.DatabaseConfig {
	t.Helper()
	if os.Getenv("ASTRO_RUNTIME_ENV") != "test" || os.Getenv("ASTRO_MARIADB_INTEGRATION") != "1" {
		t.Skip("仅在显式启用的 MariaDB 临时数据库运行")
	}
	required := func(name string) string {
		t.Helper()
		value := os.Getenv(name)
		if value == "" {
			t.Fatalf("缺少环境变量 %s", name)
		}
		return value
	}
	port, err := strconv.Atoi(required("ASTRO_DATABASE_PORT"))
	if err != nil || port < 1 || port > 65535 {
		t.Fatal("ASTRO_DATABASE_PORT 必须是有效端口")
	}
	return &config.DatabaseConfig{
		Host: required("ASTRO_DATABASE_HOST"), Port: port,
		User: required("ASTRO_DATABASE_USER"), Password: required("ASTRO_DATABASE_PASSWORD"),
		DBName: required("ASTRO_DATABASE_DBNAME"), Charset: required("ASTRO_DATABASE_CHARSET"),
	}
}

func closeDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()
	sqlDatabase, err := database.DB()
	if err != nil {
		t.Fatalf("读取数据库连接失败: %v", err)
	}
	sqlDatabase.SetMaxOpenConns(4)
	t.Cleanup(func() {
		if err := sqlDatabase.Close(); err != nil {
			t.Errorf("关闭数据库连接失败: %v", err)
		}
	})
}

func assertProjectOwnershipSchema(t *testing.T, database *gorm.DB) {
	t.Helper()
	if !database.Migrator().HasTable(&model.Project{}) || !database.Migrator().HasColumn(&model.App{}, "project_id") ||
		!database.Migrator().HasColumn(&model.App{}, "config") {
		t.Fatal("当前项目归属 schema 不完整")
	}
	columns, err := database.Migrator().ColumnTypes(&model.App{})
	if err != nil {
		t.Fatalf("读取 apps 列信息失败: %v", err)
	}
	for _, column := range columns {
		if column.Name() != "project_id" {
			continue
		}
		nullable, ok := column.Nullable()
		if !ok || nullable {
			t.Fatal("apps.project_id 必须为非空列")
		}
		if !database.Migrator().HasConstraint(&model.App{}, "Project") {
			t.Fatal("apps.project_id 缺少 Project 外键")
		}
		return
	}
	t.Fatal("未找到 apps.project_id")
}

func assertProjectOwnershipConstraints(t *testing.T) {
	t.Helper()
	userOne := model.User{Username: "integration-user-one", Password: "hash", Email: "one@example.test"}
	userTwo := model.User{Username: "integration-user-two", Password: "hash", Email: "two@example.test"}
	if err := DB.Create(&userOne).Error; err != nil {
		t.Fatalf("创建第一个测试用户失败: %v", err)
	}
	if err := DB.Create(&userTwo).Error; err != nil {
		t.Fatalf("创建第二个测试用户失败: %v", err)
	}
	projectOne := model.Project{Name: "same-name", UserID: userOne.ID, Namespace: "astro-project-integration-one"}
	projectTwo := model.Project{Name: "same-name", UserID: userTwo.ID, Namespace: "astro-project-integration-two"}
	if err := DB.Create(&projectOne).Error; err != nil {
		t.Fatalf("创建第一个项目失败: %v", err)
	}
	if err := DB.Create(&projectTwo).Error; err != nil {
		t.Fatalf("不同用户创建同名项目失败: %v", err)
	}
	if err := DB.Create(&model.Project{Name: projectOne.Name, UserID: userOne.ID, Namespace: "astro-project-duplicate-name"}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("同一用户项目重名应触发唯一约束: %v", err)
	}
	if err := DB.Create(&model.Project{Name: "other-name", UserID: userTwo.ID, Namespace: projectOne.Namespace}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("项目 Namespace 重名应触发唯一约束: %v", err)
	}
	if err := DB.Create(&model.App{Name: "orphan", Image: "nginx:alpine", Replicas: 1, ProjectID: 999999}).Error; !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("不存在的项目外键应被拒绝: %v", err)
	}

	appRepository := NewAppRepository()
	servicePort := int32(80)
	config := model.AppConfig{
		Command: []string{"/bin/demo"},
		Ports:   []model.AppPort{{Name: "http", ContainerPort: 8080, Protocol: "TCP", ServicePort: &servicePort}},
	}
	app := &model.App{Name: "same-app", Image: "nginx:alpine", Replicas: 1, ProjectID: projectOne.ID, Config: config}
	if _, err := appRepository.CreateInProject(app, userOne.ID, func(string) error {
		var visibleCount int64
		if err := DB.Model(&model.App{}).Where("id = ?", app.ID).Count(&visibleCount).Error; err != nil {
			return err
		}
		if visibleCount != 0 {
			return fmt.Errorf("事务提交前应用记录可见")
		}
		return nil
	}); err != nil {
		t.Fatalf("在项目内创建应用失败: %v", err)
	}
	storedApp, err := appRepository.GetByProjectAndID(projectOne.ID, app.ID)
	if err != nil {
		t.Fatalf("读取高级配置失败: %v", err)
	}
	if !reflect.DeepEqual(storedApp.Config, config) {
		t.Fatalf("高级配置往返不一致: got %#v, want %#v", storedApp.Config, config)
	}
	if err := DB.Model(&model.App{}).Where("id = ?", app.ID).UpdateColumn("config", nil).Error; err != nil {
		t.Fatalf("模拟旧 NULL 配置失败: %v", err)
	}
	storedApp, err = appRepository.GetByProjectAndID(projectOne.ID, app.ID)
	if err != nil || !reflect.DeepEqual(storedApp.Config, model.AppConfig{}) {
		t.Fatalf("旧 NULL 配置未归一化为空配置: app=%#v err=%v", storedApp, err)
	}
	if err := DB.Create(&model.App{Name: app.Name, Image: app.Image, Replicas: 1, ProjectID: projectOne.ID}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("同一项目应用重名应触发唯一约束: %v", err)
	}
	if err := DB.Create(&model.App{Name: app.Name, Image: app.Image, Replicas: 1, ProjectID: projectTwo.ID}).Error; err != nil {
		t.Fatalf("不同项目创建同名应用失败: %v", err)
	}
	if err := DB.Unscoped().Delete(&projectOne).Error; err == nil {
		t.Fatalf("包含应用的项目应受外键限制删除: %v", err)
	}
	var orphanCount int64
	if err := DB.Model(&model.App{}).
		Joins("LEFT JOIN projects ON projects.id = apps.project_id").
		Where("projects.id IS NULL").Count(&orphanCount).Error; err != nil {
		t.Fatalf("统计孤儿应用失败: %v", err)
	}
	if orphanCount != 0 {
		t.Fatalf("存在 %d 个孤儿应用", orphanCount)
	}
}

func assertProjectLockSerializesCreate(t *testing.T) {
	t.Helper()
	var user model.User
	if err := DB.Where("username = ?", "integration-user-one").First(&user).Error; err != nil {
		t.Fatalf("读取测试用户失败: %v", err)
	}
	project := model.Project{Name: "lock-test", UserID: user.ID, Namespace: "astro-project-lock-test"}
	if err := DB.Create(&project).Error; err != nil {
		t.Fatalf("创建锁测试项目失败: %v", err)
	}

	locked := make(chan struct{})
	release := make(chan struct{})
	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- NewProjectRepository().DeleteEmpty(project.ID, user.ID, func(string) error {
			close(locked)
			<-release
			return nil
		})
	}()
	select {
	case <-locked:
	case <-time.After(5 * time.Second):
		t.Fatal("项目删除未取得行锁")
	}

	createStarted := make(chan struct{})
	createResult := make(chan error, 1)
	callbackInvoked := false
	go func() {
		close(createStarted)
		_, err := NewAppRepository().CreateInProject(
			&model.App{Name: "blocked-app", Image: "nginx:alpine", Replicas: 1, ProjectID: project.ID},
			user.ID,
			func(string) error { callbackInvoked = true; return nil },
		)
		createResult <- err
	}()
	<-createStarted
	select {
	case err := <-createResult:
		close(release)
		t.Fatalf("应用创建未等待项目行锁: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	if err := <-deleteResult; err != nil {
		t.Fatalf("删除空项目失败: %v", err)
	}
	select {
	case err := <-createResult:
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("项目删除提交后应用创建应返回项目不存在: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("项目删除后应用创建未结束")
	}
	if callbackInvoked {
		t.Fatal("项目已删除时不应调用应用创建回调")
	}
}
