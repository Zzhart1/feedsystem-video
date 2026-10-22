package http

import (
	"context"
	"log"
	"time"

	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/config"
	"feedsystem_video_go/internal/feed"
	"feedsystem_video_go/internal/middleware/jwt"
	mqrabbit "feedsystem_video_go/internal/middleware/rabbitmq"
	"feedsystem_video_go/internal/middleware/ratelimit"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/social"
	"feedsystem_video_go/internal/video"
	"feedsystem_video_go/internal/worker"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB, cache *rediscache.Client, rmqCfg *config.RabbitMQConfig) *gin.Engine {
	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 视频/封面静态文件：URL /static/xxx → 磁盘 ./.run/uploads/xxx
	r.Static("/static", "./.run/uploads")

	accountRepo := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepo, cache)
	accountHandler := account.NewAccountHandler(accountService)

	// rate_limit
	loginLimiter := ratelimit.Limit(cache, "account_login", 10, time.Minute, ratelimit.KeyByIP)
	registerLimiter := ratelimit.Limit(cache, "account_register", 5, time.Hour, ratelimit.KeyByIP)

	likeLimiter := ratelimit.Limit(cache, "like_write", 30, time.Minute, ratelimit.KeyByAccount)
	commentLimiter := ratelimit.Limit(cache, "comment_write", 10, time.Minute, ratelimit.KeyByAccount)
	socialLimiter := ratelimit.Limit(cache, "social_write", 20, time.Minute, ratelimit.KeyByAccount)
	// 公开接口
	g := r.Group("/account")
	{
		g.POST("/register", registerLimiter, accountHandler.CreateAccount)
		g.POST("/login", loginLimiter, accountHandler.Login)
		g.POST("/refresh", accountHandler.Refresh)
		g.POST("/changePassword", accountHandler.ChangePassword)
		g.POST("/findByUsername", accountHandler.FindByUsername)
	}
	// 需要登录的接口（挂 JWT 中间件）
	protected := g.Group("")
	protected.Use(jwt.JWTAuth(accountRepo, cache))
	{
		protected.POST("/logout", accountHandler.Logout)
		protected.POST("/rename", accountHandler.Rename)
		protected.POST("/updateProfile", accountHandler.UpdateProfile)
		protected.POST("/findByID", accountHandler.FindByID) // 你的实现从登录态取 id，放这里
		protected.POST("/updateAbatar", accountHandler.UpdateAvatar)
	}

	//video

	videoRepo := video.NewVideoRepository(db)
	videoService := video.NewVideoService(videoRepo, cache, time.Minute) // 签名按你修完的来
	videoHandler := video.NewVideoHandler(videoService, accountService)
	chunkHandler := video.NewChunkUploadHandler(cache)

	videoGroup := r.Group("/video")
	{
		videoGroup.POST("/listByAuthorID", videoHandler.ListByAuthorID)
		videoGroup.POST("/getDetail", videoHandler.GetDetail)
	}
	protectedVideo := videoGroup.Group("")
	protectedVideo.Use(jwt.JWTAuth(accountRepo, cache))
	{
		protectedVideo.POST("/uploadVideo", videoHandler.UploadVideo)
		protectedVideo.POST("/uploadCover", videoHandler.UploadCover)
		protectedVideo.POST("/publish", videoHandler.PublishVideo)
		protectedVideo.POST("/delete", videoHandler.DeleteVideo)
		protectedVideo.POST("/chunk/init", chunkHandler.InitChunkUpload)
		protectedVideo.POST("/chunk/upload", chunkHandler.UploadChunk)
		protectedVideo.POST("/chunk/status", chunkHandler.ChunkStatus)
		protectedVideo.POST("/chunk/complete", chunkHandler.CompleteChunkUpload)
	}

	// ---- 阶段4：互动模块（点赞/评论/关注），RabbitMQ 连不上自动降级直写 ----
	rmq, _ := mqrabbit.NewRabbitMq(rmqCfg) // 连不上时 rmq 为 nil，下面各 MQ 全为 nil → service 降级直写
	likeMQ, _ := mqrabbit.NewLikeMQ(rmq)
	commentMQ, _ := mqrabbit.NewCommentMQ(rmq)
	socialMQ, _ := mqrabbit.NewSocialMQ(rmq)
	popularityMQ, _ := mqrabbit.NewPopularityMQ(rmq)

	// like
	likeRepository := video.NewLikeRepository(db)
	likeService := video.NewLikeService(likeRepository, videoRepo, cache, likeMQ, popularityMQ)
	likeHandler := video.NewLikeHandler(likeService)
	likeGroup := r.Group("/like")
	protectedLikeGroup := likeGroup.Group("")
	protectedLikeGroup.Use(jwt.JWTAuth(accountRepo, cache))
	{
		protectedLikeGroup.POST("/like", likeLimiter, likeHandler.Like)
		protectedLikeGroup.POST("/unlike", likeLimiter, likeHandler.Unlike)
		protectedLikeGroup.POST("/isLiked", likeHandler.IsLiked)
		protectedLikeGroup.POST("/listMyLikedVideos", likeHandler.ListMyLikedVideos)
	}

	// comment
	commentRepository := video.NewCommentRepository(db)
	commentService := video.NewCommentService(commentRepository, videoRepo, cache, commentMQ, popularityMQ)
	commentHandler := video.NewCommentHandler(commentService, accountService)
	commentGroup := r.Group("/comment")
	{
		commentGroup.POST("/listAll", commentHandler.GetAllComments)
	}
	protectedCommentGroup := commentGroup.Group("")
	protectedCommentGroup.Use(jwt.JWTAuth(accountRepo, cache))
	{
		protectedCommentGroup.POST("/publish", commentLimiter, commentHandler.PublishComment)
		protectedCommentGroup.POST("/delete", commentLimiter, commentHandler.DeleteComment)
	}

	// social
	socialRepository := social.NewSocialRepository(db)
	socialService := social.NewSocialService(socialRepository, accountRepo, socialMQ, cache)
	socialHandler := social.NewSocialHandler(socialService)
	socialGroup := r.Group("/social")
	protectedSocialGroup := socialGroup.Group("")
	protectedSocialGroup.Use(jwt.JWTAuth(accountRepo, cache))
	{
		protectedSocialGroup.POST("/follow", socialLimiter, socialHandler.Follow)
		protectedSocialGroup.POST("/unfollow", socialLimiter, socialHandler.Unfollow)
		protectedSocialGroup.POST("/getAllFollowers", socialHandler.GetAllFollowers)
		protectedSocialGroup.POST("/getAllVloggers", socialHandler.GetAllVloggers)
		protectedSocialGroup.POST("/getCounts", socialHandler.GetCounts)
	}

	// feed（阶段3：软鉴权，匿名可看，登录可选）
	feedRepo := feed.NewFeedRepository(db)
	feedService := feed.NewFeedService(feedRepo, cache)
	feedHandler := feed.NewFeedHandler(feedService)
	feedGroup := r.Group("/feed")
	feedGroup.Use(jwt.SoftJWTAuth(accountRepo, cache))
	{
		feedGroup.POST("/listLatest", feedHandler.ListLatest)
		feedGroup.POST("/listLikesCount", feedHandler.ListLikesCount)
		feedGroup.POST("/listByPopularity", feedHandler.ListByPopularity)
		feedGroup.POST("/listByTag", feedHandler.ListByTag)
		feedGroup.POST("/listByFollowing", feedHandler.ListByFollowing)
	}

	// ---- 阶段5：Outbox 事务消息 + 时间线异步化（API 进程内启动）----
	timelineMQ, err := mqrabbit.NewTimelineMQ(rmq)
	if err != nil {
		log.Printf("timelineMQ init failed (mq disabled): %v", err)
		timelineMQ = nil
	}
	worker.StartOutboxPoller(db, timelineMQ)
	worker.StartConsumer(timelineMQ, "video.timeline.update.queue", cache, rmq)

	// ---- 阶段5：SSE 通知（实时推送 + 历史拉取）----
	if rmq != nil {
		if notifCh, err := rmq.NewChannel(); err == nil {
			if err := mqrabbit.DeclareTopic(notifCh, "like.events", "notification.like", "like.like"); err != nil {
				log.Printf("notification like topic init failed: %v", err)
			}
			if err := mqrabbit.DeclareTopic(notifCh, "comment.events", "notification.comment", "comment.publish"); err != nil {
				log.Printf("notification comment topic init failed: %v", err)
			}
			if err := mqrabbit.DeclareTopic(notifCh, "social.events", "notification.social", "social.follow"); err != nil {
				log.Printf("notification social topic init failed: %v", err)
			}
			notifCh.Close()
		}
	}
	sseHub := worker.NewSSEHub(db)
	notifGroup := r.Group("/notification")
	notifGroup.Use(sseHub.SSERequireAuth())
	sseHub.RegisterRoutes(r, notifGroup)

	// 3 个 NotificationWorker（like/comment/follow），独立 Channel + 自动重连
	go func() {
		if rmq != nil {
			hub := sseHub
			ctx := context.Background()
			for _, q := range []string{"notification.like", "notification.comment", "notification.social"} {
				go func(queue string) {
					for {
						ch, err := rmq.NewChannel()
						if err != nil {
							log.Printf("notification-%s: 创建 Channel 失败: %v, 5秒后重试", queue, err)
							time.Sleep(5 * time.Second)
							continue
						}
						w := worker.NewNotificationWorker(ch, db, queue, hub)
						if err := w.Run(ctx); err != nil {
							log.Printf("notification-%s: %v, 5秒后重连...", queue, err)
						}
						ch.Close()
						time.Sleep(5 * time.Second)
					}
				}(q)
			}
		} else {
			log.Printf("Notification SSE disabled (MQ not available)")
		}
	}()

	return r
}
