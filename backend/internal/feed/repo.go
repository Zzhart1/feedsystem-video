package feed

import (
	"context"
	"feedsystem_video_go/internal/social"
	"feedsystem_video_go/internal/video"
	"time"

	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

func (repo *FeedRepository) ListLatest(ctx context.Context, limit int, latestBefore time.Time) ([]*video.Video, error) {
	var videos []*video.Video
	query := repo.db.WithContext(ctx).Model(&video.Video{}).Order("create_time DESC")
	if !latestBefore.IsZero() {
		query = query.Where("create_time<?", latestBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (repo *FeedRepository) ListLikesCountWithCursor(ctx context.Context, limit int, cursor *LikesCountCursor) ([]*video.Video, error) {
	var videos []*video.Video
	query := repo.db.WithContext(ctx).Model(&video.Video{}).Order("likes_count DESC,id DESC")
	if cursor != nil {
		query = query.Where(
			"(likes_count < ?) OR (likes_count = ? AND id < ?)",
			cursor.LikesCount, cursor.LikesCount, cursor.ID,
		)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

// TODO 阶段4：social 包写完后再补 ListByFollowing
func (repo *FeedRepository) ListByFollowing(ctx context.Context, viewerAccountID uint, limit int, timeBefore time.Time) ([]*video.Video, error) {
	var videos []*video.Video
	query := repo.db.WithContext(ctx).Model(&video.Video{}).Order("create_time DESC")
	if viewerAccountID > 0 {
		followingSubQuery := repo.db.WithContext(ctx).Model(&social.Social{}).
			Select("vlogger_id").
			Where("follower_id=?", viewerAccountID)
		query = query.Where("author_id IN (?)", followingSubQuery)
	}
	if !timeBefore.IsZero() {
		query = query.Where("create_time<?", timeBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (repo *FeedRepository) ListByPopularity(ctx context.Context, limit int, popularityBefore int64, timeBefore time.Time, idBefore uint) ([]*video.Video, error) {
	var videos []*video.Video
	query := repo.db.WithContext(ctx).Model(&video.Video{}).Order("popularity DESC,create_time DESC,id DESC")
	if !timeBefore.IsZero() && idBefore > 0 {
		query = query.Where(
			"(popularity < ?) OR (popularity = ? AND create_time < ?) OR (popularity = ? AND create_time = ? AND id < ?)",
			popularityBefore,
			popularityBefore, timeBefore,
			popularityBefore, timeBefore, idBefore,
		)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (repo *FeedRepository) GetByIDs(ctx context.Context, ids []uint) ([]*video.Video, error) {
	var videos []*video.Video
	if len(ids) == 0 {
		return videos, nil
	}
	if err := repo.db.WithContext(ctx).Where("id IN ?", ids).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (repo *FeedRepository) ListByTag(ctx context.Context, tagName string, limit int) ([]*video.Video, error) {
	var videos []*video.Video
	err := repo.db.WithContext(ctx).Model(&video.Video{}).Table("videos").
		Joins("JOIN video_tags ON video_tags.video_id = videos.id").
		Joins("JOIN tags ON tags.id = video_tags.tag_id").
		Where("tags.name = ?", tagName).
		Order("videos.create_time desc").
		Limit(limit).
		Find(&videos).Error
	return videos, err
}

// BatchGetLikedVideoIDs 返回当前用户在给定视频集合中已点赞的 videoID 集合
func (repo *FeedRepository) BatchGetLikedVideoIDs(ctx context.Context, accountID uint, videoIDs []uint) (map[uint]bool, error) {
	result := make(map[uint]bool)
	if accountID == 0 || len(videoIDs) == 0 {
		return result, nil
	}
	var likedIDs []uint
	err := repo.db.WithContext(ctx).
		Table("likes").
		Where("account_id = ? AND video_id IN ?", accountID, videoIDs).
		Pluck("video_id", &likedIDs).Error
	if err != nil {
		return result, err
	}
	for _, id := range likedIDs {
		result[id] = true
	}
	return result, nil
}
