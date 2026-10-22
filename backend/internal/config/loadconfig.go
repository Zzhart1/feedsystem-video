package config

import (
	"os"
	"strconv"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type Config struct {
	Server   struct{ Port int }
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
}

// Load 先给默认值，再用环境变量覆盖（阶段 2 可升级成 yaml 配置文件）
func Load() Config {
	var cfg Config
	cfg.Server.Port = 8080
	cfg.Database = DatabaseConfig{
		Host: "localhost", Port: 3306,
		User: "root", Password: "3295", // ← 改成你 MySQL root 的真实密码
		DBName: "feedsystem",
	}
	cfg.Redis = RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "123456",
		DB:       0,
	}
	cfg.RabbitMQ = RabbitMQConfig{
		Host:     "localhost",
		Port:     5672,
		Username: "admin",
		Password: "password123",
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	return cfg
}
