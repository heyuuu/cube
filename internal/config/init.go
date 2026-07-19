package config

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/heyuuu/cube/internal/model"
	"github.com/heyuuu/cube/internal/util/pathkit"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 默认配置目录
const defaultCfgPath = "~/.go-cube/"

func InitConfig(cfgPath string) {
	if len(cfgPath) == 0 {
		cfgPath = defaultCfgPath
	}
	cfgPath = pathkit.RealPath(cfgPath)

	// 记录日志目录
	configPath = cfgPath

	// 初始化配置文件 config.json
	err := initDefaultConf(cfgPath)
	if err != nil {
		log.Fatalln(err)
	}

	// 初始化数据文件 data.db
	err = initDefaultDb(cfgPath)
	if err != nil {
		log.Fatalln(err)
	}
}

// config path
var configPath string

func ConfigPath() string {
	return configPath
}

// config file (config.json)
var defaultConf Config

func Default() Config {
	return defaultConf
}

func initDefaultConf(cfgPath string) error {
	cfgFile := filepath.Join(cfgPath, "config.json")
	// 若配置文件不存在则跳过
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		return nil
	}
	return parseConfigFile(cfgFile, &defaultConf)
}

func parseConfigFile(cfgFile string, cfg *Config) error {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("read config file failed: %w", err)
	}

	err = json.Unmarshal(data, cfg)
	if err != nil {
		return fmt.Errorf("unmarshal config data failed: %w", err)
	}

	return nil
}

// sqlite file (data.db)

var dataDb *gorm.DB

func DataDb() *gorm.DB {
	return dataDb
}

func initDefaultDb(cfgPath string) error {
	dbFile := filepath.Join(cfgPath, "data.db")

	db, err := initDb(dbFile)
	if err != nil {
		return err
	}

	dataDb = db
	return nil
}

func initDb(dsn string) (*gorm.DB, error) {
	// 连接到 SQLite 数据库
	slog.Info("init db", "dsn", dsn)

	gormConfig := &gorm.Config{}
	if IsDebug() {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}
	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	// 自动迁移数据表结构
	err = db.AutoMigrate(
		&model.ProjectSelectLog{},
		&model.ProjectOpenLog{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据表结构迁移失败: %w", err)
	}

	return db, nil
}
