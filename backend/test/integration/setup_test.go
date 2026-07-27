//go:build integration

package integration

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tcm-history-ai/backend/pkg/config"
	"tcm-history-ai/backend/pkg/idgen"
)

// testConfig 是 config.test.yaml 的 typed 视图。仅捕获集成测试需要的字段。
//
// 注意：本文件**不导入任何 *-service/internal/... 包**，因为 Go 的 internal
// 可见性规则禁止跨服务引用。所有 schema 与种子数据通过读取各服务 migrations/
// 下的 .up.sql 文件直接执行；所有数据写入/查询通过 raw SQL 完成。
type testConfig struct {
	DB       dbConfig        `mapstructure:"db"`
	RabbitMQ rabbitMQConfig  `mapstructure:"rabbitmq"`
	JWT      jwtConfig       `mapstructure:"jwt"`
	Milvus   externalService `mapstructure:"milvus"`
	Neo4j    externalService `mapstructure:"neo4j"`
}

type dbConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type rabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
	Exchange string `mapstructure:"exchange"`
}

type jwtConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type externalService struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Enabled  bool   `mapstructure:"enabled"`
}

// 全局共享状态。在 TestMain 中初始化；每条用例直接读取这些变量。
var (
	cfg     testConfig
	db      *gorm.DB
	skipAll bool // 当 PostgreSQL / RabbitMQ 不可达时为 true
)

// cfgFile 是命令行 -config 参数，允许覆盖配置文件路径。
var cfgFile = flag.String("config", "testdata/config.test.yaml", "Path to integration test config file")

// migrationsRoot 是各服务 migrations 目录的相对路径前缀。
// 测试在 backend/test/integration/ 下执行，因此相对路径为 ../../../<svc>/migrations。
const migrationsRoot = "../../"

// TestMain 负责加载配置、等待依赖就绪、应用迁移、初始化 idgen。
//
// 任一关键依赖不可达时跳过整个套件（os.Exit(0)），而不是失败——
// 这样本地无依赖环境运行 `go test -tags=integration` 不会报错。
func TestMain(m *testing.M) {
	flag.Parse()
	if err := config.Load(*cfgFile, &cfg); err != nil {
		log.Fatalf("integration: load config %s: %v", *cfgFile, err)
	}
	idgen.Init(1)

	// 1. 等待 PostgreSQL 就绪；不可达则跳过全部测试。
	if err := waitPostgresReady(15 * time.Second); err != nil {
		log.Printf("integration: PostgreSQL not ready, skipping suite: %v", err)
		skipAll = true
		os.Exit(0)
	}
	gormDB, err := openGORM()
	if err != nil {
		log.Fatalf("integration: open gorm: %v", err)
	}
	db = gormDB

	// 2. 等待 RabbitMQ 就绪；不可达也跳过全部测试（graph sync 用例需要）。
	if err := waitRabbitMQReady(15 * time.Second); err != nil {
		log.Printf("integration: RabbitMQ not ready, skipping suite: %v", err)
		skipAll = true
		os.Exit(0)
	}

	// 3. 应用各服务的 migrations/*.up.sql，建立全部 schema + 种子数据。
	//    migrations 文件本身就是生产 DDL，复用它们能保证测试 schema 与生产一致，
	//    同时避免在测试代码里重复维护建表语句、也无需导入服务 internal 实体类型。
	if err := applyAllMigrations(db); err != nil {
		log.Fatalf("integration: apply migrations: %v", err)
	}

	// 4. 清空业务表，确保每轮 CI 起点干净（保留 roles / permissions 种子）。
	if err := truncateBusinessTables(db); err != nil {
		log.Fatalf("integration: truncate tables: %v", err)
	}

	os.Exit(m.Run())
}

// waitPostgresReady 轮询 TCP 端口 + 真实 SQL PING，直到 PostgreSQL 可用或超时。
func waitPostgresReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	dsn := postgresDSN()
	var lastErr error
	for time.Now().Before(deadline) {
		gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			sqlDB, pingErr := gdb.DB()
			if pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					_ = sqlDB.Close()
					return nil
				}
			}
			if sqlDB != nil {
				_ = sqlDB.Close()
			}
			lastErr = pingErr
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("postgres not ready after %s: %w", timeout, lastErr)
}

// waitRabbitMQReady 轮询 RabbitMQ amqp 端口，直到可拨通或超时。
func waitRabbitMQReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	vhost := cfg.RabbitMQ.VHost
	if vhost == "" {
		vhost = "/"
	}
	uri := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port, vhost)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := amqp091.DialConfig(uri, amqp091.Config{
			Heartbeat: 5 * time.Second,
			Locale:    "en_US",
		})
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("rabbitmq not ready after %s: %w", timeout, lastErr)
}

// postgresDSN 拼接 PostgreSQL DSN 字符串。
func postgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.DBName, cfg.DB.SSLMode,
	)
}

// openGORM 打开共享的 *gorm.DB。集成测试期间所有用例共用同一连接。
func openGORM() (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(postgresDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	return gdb, nil
}

// applyAllMigrations 按服务顺序读取各 migrations/*.up.sql 并执行。
// 顺序：user-service → learning-service → knowledge-service → graph-service。
//   - user-service 先建，因为其 roles / permissions 是其他服务种子数据可能引用的依赖
//   - knowledge-service 的 document_chunks FK 引用 documents
//   - graph-service 完全独立
//
// 注意：seed_learning / seed_core_nodes 等 seed 类迁移会被执行——
// 它们均使用 ON CONFLICT DO NOTHING / NOT EXISTS 防重，因此重复执行是安全的；
// 但 truncateBusinessTables 会清空业务数据，保留 roles / permissions。
func applyAllMigrations(gdb *gorm.DB) error {
	services := []string{
		"user-service",
		"learning-service",
		"knowledge-service",
		"graph-service",
	}
	for _, svc := range services {
		dir := filepath.Join(migrationsRoot, svc, "migrations")
		files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
		if err != nil {
			return fmt.Errorf("glob migrations for %s: %w", svc, err)
		}
		sort.Strings(files) // 按文件名前缀数字顺序执行
		for _, f := range files {
			sql, err := os.ReadFile(f)
			if err != nil {
				return fmt.Errorf("read %s: %w", f, err)
			}
			if err := gdb.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("exec %s: %w", f, err)
			}
		}
	}
	return nil
}

// truncateBusinessTables 清空所有业务表数据，但保留表结构与种子角色/权限。
// 每轮 CI 在 TestMain 中调用一次；单个测试内部不应再调用以避免相互干扰。
//
// 顺序：先清子表（有外键依赖的），再清父表。CASCADE 兜底处理未列出的依赖。
func truncateBusinessTables(gdb *gorm.DB) error {
	tables := []string{
		// graph-service
		"graph_sync_logs",
		"graph_edges",
		"graph_nodes",
		// knowledge-service
		"rag_queries",
		"embedding_tasks",
		"document_chunks",
		"documents",
		// learning-service
		"learning_wrong_questions",
		"learning_exam_attempts",
		"learning_questions",
		"learning_exams",
		"learning_records",
		"learning_enrollments",
		"learning_lessons",
		"learning_courses",
		"learning_study_plans",
		// user-service 业务表（保留 roles / permissions / role_permissions 种子）
		"tenant_members",
		"user_settings",
		"user_profiles",
		"user_roles",
		"users",
		"tenants",
	}
	for _, t := range tables {
		// RESTART IDENTITY 重置 SERIAL 序列；CASCADE 清除外键依赖行。
		if err := gdb.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", t)).Error; err != nil {
			return fmt.Errorf("truncate %s: %w", t, err)
		}
	}
	return nil
}

// skipIfNoDeps 是每个测试函数开头的护栏：当 TestMain 跳过依赖初始化时，
// 单个测试也会快速 skip，避免对 nil db 的访问。
func skipIfNoDeps(t *testing.T) {
	t.Helper()
	if skipAll || db == nil {
		t.Skip("integration: dependencies (PostgreSQL/RabbitMQ) not available")
	}
}

// resetTablesInTest 清空指定表，供单条用例在子测试开始前重置自己的数据域。
// 与 truncateBusinessTables 不同，本函数不重置整个 schema，仅清空传入的表。
func resetTablesInTest(t *testing.T, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tbl)).Error; err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}

// newContext 返回带 30s 超时的 context，便于在用例中统一控制。
func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// nextID 返回一个雪花 ID，便于用例在 raw SQL INSERT 时显式指定主键。
// idgen 已在 TestMain 中 Init。
func nextID() int64 { return idgen.Next() }
