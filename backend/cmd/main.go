package main

import (
	"log"
	"strconv"

	"feedsystem_video_go/internal/config"
	"feedsystem_video_go/internal/db"
	apphttp "feedsystem_video_go/internal/http"
	rediscache "feedsystem_video_go/internal/middleware/redis"
)

func main() {
	cfg := config.Load()

	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(sqlDB); err != nil {
		log.Fatalf("Failed to auto migrate: %v", err)
	}
	defer db.CloseDB(sqlDB)

	// 阶段 2 之前不接 Redis：传 nil，service 内部自动降级
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to init redis: %v", err)
	}
	
	r := apphttp.SetRouter(sqlDB, cache, &cfg.RabbitMQ)
	log.Printf("Server is running on port %d", cfg.Server.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
