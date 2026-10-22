package feed

import (
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	service *FeedService
}

func NewFeedHandler(service *FeedService) *FeedHandler {
	return &FeedHandler{service: service}
}

func (f *FeedHandler) ListLatest(c *gin.Context) {
	var req ListLatestRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		viewerAccountID = 0
	}
	var latestTime time.Time
	if req.LatestTime > 0 {
		latestTime = time.UnixMilli(req.LatestTime)
	}
	feedItems, err := f.service.ListLatest(c.Request.Context(), req.Limit, latestTime, viewerAccountID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	feedItems.VideoList = f.nonNilFeedVideoItems(feedItems.VideoList)
	c.JSON(200, feedItems)
}

func (f *FeedHandler) ListLikesCount(c *gin.Context) {
	var req ListLikesCountRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		viewerAccountID = 0
	}
	var cursor *LikesCountCursor
	if req.LikesCountBefore != nil || req.IDBefore != nil {
		if req.LikesCountBefore == nil || req.IDBefore == nil {
			c.JSON(400, gin.H{"error": "likes_count_before and id_before must be provided together"})
			return
		}

		likesCountBefore := *req.LikesCountBefore
		idBefore := *req.IDBefore

		if likesCountBefore < 0 {
			c.JSON(400, gin.H{"error": "invalid cursor: likes_count_before must be >= 0"})
			return
		}
		if idBefore == 0 {
			if likesCountBefore != 0 {
				c.JSON(400, gin.H{"error": "invalid cursor: id_before must be > 0"})
				return
			}
		} else {
			cursor = &LikesCountCursor{
				LikesCount: likesCountBefore,
				ID:         idBefore,
			}
		}
	}
	viewerAccountID, err = jwt.GetAccountID(c)
	if err != nil {
		viewerAccountID = 0
	}
	feedItems, err := f.service.ListLikesCount(c.Request.Context(), req.Limit, cursor, viewerAccountID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	feedItems.VideoList = f.nonNilFeedVideoItems(feedItems.VideoList)
	c.JSON(200, feedItems)
}

// TODO 阶段4：social 模块就绪后，实现 ListByFollowing Handler（关注流）
func (f *FeedHandler) ListByFollowing(c *gin.Context) {
	var req ListByFollowingRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	var latestBefore time.Time
	if req.LatestTime > 0 {
		latestBefore = time.UnixMilli(req.LatestTime)
	}
	feedItems, err := f.service.ListByFollowing(c.Request.Context(), req.Limit, viewerAccountID, latestBefore)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	feedItems.VideoList = f.nonNilFeedVideoItems(feedItems.VideoList)
	c.JSON(200, feedItems)
}

func (f *FeedHandler) ListByPopularity(c *gin.Context) {
	var req ListByPopularityRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		viewerAccountID = 0
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	var latestIDBefore uint
	var latestBefore time.Time
	var latestPopularity int64
	anyCursor := !req.LatestBefore.IsZero() || req.LatestIDBefore != nil
	if anyCursor {
		if req.LatestBefore.IsZero() || req.LatestIDBefore == nil || *req.LatestIDBefore == 0 {
			c.JSON(400, gin.H{"error": "latest_before and latest_id_before must be provided together"})
			return
		}
		latestBefore = req.LatestBefore
		latestIDBefore = *req.LatestIDBefore
		latestPopularity = req.LatestPopularity
	}
	feedItems, err := f.service.ListByPopularity(
		c.Request.Context(),
		req.Limit,
		req.AsOf,
		req.Offset,
		viewerAccountID,
		latestPopularity,
		latestBefore,
		latestIDBefore,
	)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	feedItems.VideoList = f.nonNilFeedVideoItems(feedItems.VideoList)
	c.JSON(200, feedItems)
}

func (f *FeedHandler) nonNilFeedVideoItems(items []FeedVideoItem) []FeedVideoItem {
	if items == nil {
		return []FeedVideoItem{}
	}
	return items
}

func (f *FeedHandler) ListByTag(c *gin.Context) {
	var req struct {
		TagName string `json:"tag_name"`
		Limit   int    `json:"limit"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), err.Error())
		return
	}
	if req.TagName == "" {
		c.JSON(400, gin.H{"error": "tag_name is required"})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerAccountID, _ := jwt.GetAccountID(c)
	items, err := f.service.ListByTag(c.Request.Context(), req.TagName, req.Limit, viewerAccountID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"video_list": f.nonNilFeedVideoItems(items)})
}
