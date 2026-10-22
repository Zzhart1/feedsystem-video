package db

import (
	"fmt"

	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/config"
	"feedsystem_video_go/internal/social"
	"feedsystem_video_go/internal/video"
	"feedsystem_video_go/internal/worker"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	// 以后每加一个模块，就在这追加一个 &xxx.Xxx{}
	return db.AutoMigrate(
		&account.Account{},
		&video.Video{},
		&video.Tag{},
		&video.VideoTag{},
		&video.Like{},
		&video.Comment{},
		&social.Social{},
		&video.OutboxMsg{},
		&worker.Notification{},
	)
}

func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
