package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"feedsystem_video_go/internal/apierror"
	rediscache "feedsystem_video_go/internal/middleware/redis"

	"gorm.io/gorm"
)

type VideoService struct {
	repo     *VideoRepository
	cache    *rediscache.Client
	cacheTTL time.Duration
}

func NewVideoService(repo *VideoRepository, cache *rediscache.Client, cacheTTL time.Duration) *VideoService {
	return &VideoService{repo: repo, cache: cache, cacheTTL: cacheTTL}
}

func (vs *VideoService) Publish(ctx context.Context, video *Video, tagsRaw string) error {
	if video == nil {
		return errors.New("video is nil")
	}

	video.Title = strings.TrimSpace(video.Title)
	video.PlayURL = strings.TrimSpace(video.PlayURL)
	video.CoverURL = strings.TrimSpace(video.CoverURL)

	if video.Title == "" {
		return errors.New("title is required")
	}
	if video.PlayURL == "" {
		return errors.New("playurl is required")
	}
	if video.CoverURL == "" {
		return errors.New("cover url is required")
	}

	// 解析标签：优先用传入的 tags 字段（逗号分隔），再从标题/描述里提取 #tag
	tagSet := make(map[string]bool)
	for _, t := range strings.Split(tagsRaw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tagSet[t] = true
		}
	}
	for _, t := range ExtractTags(video.Title + " " + video.Description) {
		tagSet[t] = true
	}

	err := vs.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(video).Error; err != nil {
			return err
		}

		outboxmsg := &OutboxMsg{
			VideoID:   video.ID,
			EventType: "publish",
			Status:    "pending",
		}

		if err := tx.Create(outboxmsg).Error; err != nil {
			return err
		}
		for tagName := range tagSet {
			var tag Tag
			if err := tx.Where("name=?", tagName).FirstOrCreate(&tag, Tag{Name: tagName}).Error; err != nil {
				return err
			}
			videotag := &VideoTag{
				VideoID: video.ID,
				TagID:   tag.ID,
			}
			if err := tx.Create(videotag).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (vs *VideoService) Delete(ctx context.Context, id uint, authorID uint) error {
	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if video == nil {
		return errors.New("video not founded")
	}
	if video.AuthorID != authorID {
		return apierror.ErrUnauthorized
	}
	if err := vs.repo.DeleteVideo(ctx, id); err != nil {
		return err
	}
	if vs.cache != nil {
		// 清详情缓存 + 实体缓存 + 从全局时间线 ZSET 移除
		_ = vs.cache.Del(context.Background(), vs.cache.Key("video:detail:id=%d", id))
		_ = vs.cache.Del(context.Background(), vs.cache.Key("video:entity:%d", id))
		_ = vs.cache.ZRem(context.Background(), vs.cache.Key("feed:global_timeline"), fmt.Sprintf("%d", id))
	}
	return nil
}

func (vs *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	videos, err := vs.repo.ListByAuthorID(ctx, int64(authorID))
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (vs *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	cacheKey := vs.cache.Key("video:detail:id=%d", id)

	getCache := func() (*Video, bool) {
		opctx, opcancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer opcancel()
		b, err := vs.cache.GetBytes(opctx, cacheKey)
		if err != nil {
			return nil, false
		}
		var video Video
		if err := json.Unmarshal(b, &video); err != nil {
			return nil, false
		}
		return &video, true
	}

	setCache := func(video *Video) {
		b, err := json.Marshal(video)
		if err != nil {
			return
		}
		opctx, opcancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer opcancel()
		vs.cache.SetBytes(opctx, cacheKey, b, vs.cacheTTL)
	}
	if vs.cache != nil {
		video, ok := getCache()
		if ok {
			return video, nil
		}
		opctx, opcancel := context.WithTimeout(ctx, 50*time.Millisecond)
		b, err := vs.cache.GetBytes(opctx, cacheKey)
		opcancel()
		if err == nil {
			var video Video
			err := json.Unmarshal(b, &video)
			if err == nil {
				return &video, nil
			}
		} else if rediscache.IsMiss(err) {
			lockKey := "lock:" + cacheKey
			lockctx, lockcancel := context.WithTimeout(ctx, 50*time.Millisecond)
			token, locked, err := vs.cache.Lock(lockctx, lockKey, 2*time.Second)
			lockcancel()
			if err == nil && locked {
				defer func() { _ = vs.cache.Unlock(ctx, lockKey, token) }()
				if v, ok := getCache(); ok {
					return v, nil
				}
				v, err := vs.repo.GetByID(ctx, id)
				if err != nil {
					return nil, err
				}
				setCache(v)
				return v, nil
			}
			for i := 0; i < 5; i++ {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(50 * time.Millisecond):
				}
				if v, ok := getCache(); ok {
					return v, nil
				}
			}
		}
	}
	//从数据库中找
	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if vs.cache != nil {
		setCache(video)
	}
	return video, nil
}

func (vs *VideoService) UpdateLikesCount(ctx context.Context, id uint, likesCount int64) error {
	if err := vs.repo.UpdateLikesCount(ctx, id, likesCount); err != nil {
		return err
	}
	return nil
}
