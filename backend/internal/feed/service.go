package feed

import (
	"context"
	"encoding/json"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/video"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type FeedService struct {
	repo         *FeedRepository
	rediscache   *rediscache.Client
	localcache   *cache.Cache
	cacheTTL     time.Duration
	requestGroup singleflight.Group
}

type CachedFeedData struct {
	PublicVideos []video.Video `json:"public_videos"`
}

func NewFeedService(repo *FeedRepository, rediscache *rediscache.Client) *FeedService {
	return &FeedService{repo: repo, rediscache: rediscache, localcache: cache.New(3*time.Second, 5*time.Second), cacheTTL: 24 * time.Hour}
}

func (f *FeedService) GetVideoByIDs(ctx context.Context, videoIDs []uint) ([]*video.Video, error) {
	//三级缓存

	//localcache
	if len(videoIDs) == 0 {
		return []*video.Video{}, nil
	}
	var missed1 []uint
	m := make(map[uint]*video.Video)
	for _, id := range videoIDs {
		cacheKey := f.rediscache.Key("video:entity:%d", id)
		if f.localcache != nil {
			if v, found := f.localcache.Get(cacheKey); found {
				if item, ok := v.(video.Video); ok {
					m[id] = &item
					continue
				}
			}
		}
		missed1 = append(missed1, id)
	}
	if len(missed1) == 0 {
		return buildOrderedResult(videoIDs, m), nil
	}

	//rediscache
	var missed2 []uint
	if f.rediscache != nil {
		if len(missed1) > 0 {
			cacheKeys := make([]string, len(missed1))
			for i, id := range missed1 {
				cacheKeys[i] = f.rediscache.Key("video:entity:%d", id)
			}
			opctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			results, err := f.rediscache.MGet(opctx, cacheKeys...)
			cancel()
			if err == nil {
				for i, b := range results {
					id := missed1[i]
					if b != nil {
						if str, ok := b.(string); ok {
							var data video.Video
							err := json.Unmarshal([]byte(str), &data)
							if err == nil {
								m[id] = &data
								// 回写更新 L1 本地缓存
								if f.localcache != nil {
									f.localcache.Set(cacheKeys[i], data, 5*time.Second)
								}
								continue
							}
						}
					}
					missed2 = append(missed2, id)
				}
			} else {
				// 如果 Redis 挂了或者超时了，全部降级到 L3
				missed2 = missed1
				log.Printf("L2 Redis MGet 失败，全部降级到 MySQL: %v", err)
			}
		}
	} else {
		missed2 = missed1
	}

	if len(missed2) == 0 {
		return buildOrderedResult(videoIDs, m), nil
	}

	//mysql
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, id := range missed2 {
		wg.Add(1)
		go func(videoID uint) {
			defer wg.Done()
			sfKey := f.rediscache.Key("sf:entiry:%d", videoID)
			v, err, _ := f.requestGroup.Do(sfKey, func() (interface{}, error) {
				videoList, err := f.repo.GetByIDs(ctx, []uint{videoID})
				if err != nil {
					return nil, err
				}
				videoCopy := *videoList[0]
				cacheKey := f.rediscache.Key("video:entity:%d", videoID)
				if f.rediscache != nil {
					b, err := json.Marshal(videoCopy)
					if err == nil {
						go func(k string, b []byte) {
							opctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
							defer cancel()
							f.rediscache.SetBytes(opctx, k, b, time.Hour)
						}(cacheKey, b)
					}
				}
				return videoList[0], nil
			})
			if err == nil && v != nil {
				safeCopy := *(v.(*video.Video))
				mu.Lock()
				m[id] = &safeCopy
				mu.Unlock()
				f.localcache.Set(f.rediscache.Key("video:entity:%d", safeCopy.ID), safeCopy, 5*time.Second)
			}
		}(id)
	}
	wg.Wait()
	return buildOrderedResult(videoIDs, m), nil
}

// 冷热分离+游标分页
func (f *FeedService) ListLatest(ctx context.Context, limit int, latestBefore time.Time, viewerAccountID uint) (ListLatestResponse, error) {
	if f.rediscache == nil {
		return f.listLatestFromDB(ctx, limit, latestBefore, viewerAccountID)
	}
	zsetTail, err := f.rediscache.ZRangeWithScores(ctx, f.rediscache.Key("feed:global_timeline"), 0, 0)

	if err != nil {
		return f.listLatestFromDB(ctx, limit, latestBefore, viewerAccountID)
	}

	isEmpty := len(zsetTail) == 0
	if isEmpty {
		sfKey := f.rediscache.Key("sf:fallback:global_timeline_rebuild")
		v, err, _ := f.requestGroup.Do(sfKey, func() (interface{}, error) {
			dbVideos, err := f.repo.ListLatest(ctx, limit, latestBefore)
			if err != nil {
				return nil, err
			}
			if len(dbVideos) == 0 {
				return "EMPTY_DB", nil
			}
			//重建zset
			bgCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			var zElements []redis.Z
			for _, video := range dbVideos {
				zElements = append(zElements, redis.Z{
					Score:  float64(video.CreateTime.UnixMilli()),
					Member: fmt.Sprintf("%d", video.ID),
				})
			}
			f.rediscache.ZAdd(bgCtx, f.rediscache.Key("feed:global_timeline"), zElements...)
			return dbVideos, nil
		})
		if err != nil {
			return ListLatestResponse{}, err
		}
		if v == "EMPTY_DB" {
			return ListLatestResponse{HasMore: false}, nil
		}
		f.ListLatest(ctx, limit, latestBefore, viewerAccountID)
	}
	waterMark := int64(zsetTail[0].Score)
	reqTime := time.Now().UnixMilli()
	if !latestBefore.IsZero() {
		reqTime = latestBefore.UnixMilli()
	}
	var baseVideos []*video.Video
	if reqTime <= waterMark {
		//冷数据
		//针对个别用户的防并发
		sfKey := f.rediscache.Key("sf:cold:listLatest:%d:%d", limit, reqTime)
		v, err, _ := f.requestGroup.Do(sfKey, func() (interface{}, error) {
			return f.repo.ListLatest(ctx, limit, latestBefore)
		})
		if err != nil {
			return ListLatestResponse{}, err
		}
		baseVideos = v.([]*video.Video)
	} else {
		//热数据
		maxScore := "+inf"
		if !latestBefore.IsZero() {
			maxScore = fmt.Sprintf("%d", reqTime-1)
		}
		videoIDsStr, err := f.rediscache.ZRevRangeByScore(ctx, f.rediscache.Key("feed:global_timeline"), maxScore, "-inf", 0, int64(limit))
		if err != nil {
			return ListLatestResponse{}, err
		}
		var ids []uint
		for _, idStr := range videoIDsStr {
			if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
				ids = append(ids, uint(id))
			}
		}
		if len(ids) > 0 {
			baseVideos, err = f.GetVideoByIDs(ctx, ids)
			if err != nil {
				return ListLatestResponse{}, err
			}
		}

		//刚好击穿冷热边界
		if len(baseVideos) < limit {
			reqCounts := limit - len(baseVideos)
			var reqTime time.Time
			if len(baseVideos) > 0 {
				reqTime = baseVideos[len(baseVideos)-1].CreateTime
			} else {
				reqTime = latestBefore
			}
			sfKey := f.rediscache.Key("sf:stitch:listLatest:%d:%d", reqCounts, reqTime.UnixMilli())
			v, err, _ := f.requestGroup.Do(sfKey, func() (interface{}, error) {
				return f.repo.ListLatest(ctx, reqCounts, reqTime)
			})
			if err == nil {
				coldVideos := v.([]*video.Video)
				baseVideos = append(baseVideos, coldVideos...)
			}
		}
	}
	var nextTime int64
	if len(baseVideos) > 0 {
		nextTime = baseVideos[len(baseVideos)-1].CreateTime.UnixMilli()
	}
	hasMore := len(baseVideos) == limit
	feedVideoItems, err := f.buildFeedVideos(ctx, baseVideos, viewerAccountID)
	if err != nil {
		return ListLatestResponse{}, err
	}
	return ListLatestResponse{
		VideoList: feedVideoItems,
		NextTime:  nextTime,
		HasMore:   hasMore,
	}, nil
}

func (f *FeedService) listLatestFromDB(ctx context.Context, limit int, latestBefore time.Time, viewerAccountID uint) (ListLatestResponse, error) {
	videos, err := f.repo.ListLatest(ctx, limit, latestBefore)
	if err != nil {
		return ListLatestResponse{}, err
	}
	var nextTime int64
	if len(videos) > 0 {
		nextTime = videos[len(videos)-1].CreateTime.UnixMilli()
	}
	hasMore := len(videos) == limit
	feedVideoItems, err := f.buildFeedVideos(ctx, videos, viewerAccountID)
	if err != nil {
		return ListLatestResponse{}, err
	}
	return ListLatestResponse{
		VideoList: feedVideoItems,
		NextTime:  nextTime,
		HasMore:   hasMore,
	}, nil
}

func (f *FeedService) ListLikesCount(ctx context.Context, limit int, cursor *LikesCountCursor, viewerAccountID uint) (ListLikesCountResponse, error) {
	videos, err := f.repo.ListLikesCountWithCursor(ctx, limit, cursor)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	hasMore := len(videos) == limit
	feedVideoItems, err := f.buildFeedVideos(ctx, videos, viewerAccountID)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	resp := ListLikesCountResponse{
		VideoList: feedVideoItems,
		HasMore:   hasMore,
	}
	if len(videos) > 0 {
		last := videos[len(videos)-1]
		nextLikesCounts := last.LikesCount
		nextID := last.ID
		resp.NextIDBefore = &nextID
		resp.NextLikesCountBefore = &nextLikesCounts
	}
	return resp, nil
}

// TODO 阶段4：social 模块就绪后，实现 ListByFollowing（关注流：DB 查询 + Redis 缓存 + 防击穿锁）
func (f *FeedService) ListByFollowing(ctx context.Context, limit int, viewerAccountID uint, latestBefore time.Time) (ListByFollowingResponse, error) {
	doListFromDB := func() (ListByFollowingResponse, error) {
		videos, err := f.repo.ListByFollowing(ctx, viewerAccountID, limit, latestBefore)
		if err != nil {
			return ListByFollowingResponse{}, err
		}
		var nextTime time.Time
		if len(videos) > 0 {
			nextTime = videos[len(videos)-1].CreateTime
		}
		hasMore := len(videos) == limit
		feedVideoItems, err := f.buildFeedVideos(ctx, videos, viewerAccountID)
		if err != nil {
			return ListByFollowingResponse{}, err
		}
		return ListByFollowingResponse{
			VideoList: feedVideoItems,
			NextTime:  nextTime.Unix(),
			HasMore:   hasMore,
		}, nil
	}
	var cacheKey string
	if viewerAccountID != 0 && f.rediscache != nil {
		before := int64(0)
		if !latestBefore.IsZero() {
			before = latestBefore.Unix()
		}
		cacheKey = f.rediscache.Key("feed:listByFollowing:limit=%d:accountID=%d:before=%d", limit, viewerAccountID, before)
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		b, err := f.rediscache.GetBytes(cacheCtx, cacheKey)
		if err == nil {
			var cached ListByFollowingResponse
			if err := json.Unmarshal(b, &cached); err == nil {
				return cached, nil
			}
		} else if rediscache.IsMiss(err) {
			lockKey := "lock:" + cacheKey
			token, locked, _ := f.rediscache.Lock(ctx, lockKey, 500*time.Millisecond)
			if locked {
				defer func() { _ = f.rediscache.Unlock(context.Background(), lockKey, token) }()
				b, err := f.rediscache.GetBytes(cacheCtx, cacheKey)
				if err == nil {
					var cached ListByFollowingResponse
					if err := json.Unmarshal(b, &cached); err == nil {
						return cached, nil
					}
				}
				resp, err := doListFromDB()
				if err != nil {
					return ListByFollowingResponse{}, err
				} else {
					if f.rediscache != nil {
						if b, err := json.Marshal(resp); err == nil {
							_ = f.rediscache.SetBytes(ctx, cacheKey, b, f.cacheTTL)
						}
					}
				}
				return resp, nil
			} else {
				for i := 0; i < 5; i++ {
					time.Sleep(20 * time.Millisecond)
					if b, err := f.rediscache.GetBytes(cacheCtx, cacheKey); err == nil {
						var cached ListByFollowingResponse
						if err := json.Unmarshal(b, &cached); err == nil {
							return cached, nil
						}
					}
				}
			}
		}
	}
	resp, err := doListFromDB()
	if err != nil {
		return ListByFollowingResponse{}, err
	}
	if cacheKey != "" {
		if b, err := json.Marshal(resp); err == nil {
			cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			_ = f.rediscache.SetBytes(cacheCtx, cacheKey, b, f.cacheTTL)
		}
	}
	return resp, nil
}

func (f *FeedService) ListByPopularity(ctx context.Context, limit int, reqAsof int64, offset int, viewerAccountID uint, latestPopuLarity int64, latestBefore time.Time, latestIDBefore uint) (ListByPopularityResponse, error) {
	//redis热榜（稳定分页asOf,offset)
	if f.rediscache != nil {
		asOf := time.Now().UTC().Truncate(time.Minute)
		if reqAsof > 0 {
			asOf = time.Unix(reqAsof, 0).UTC().Truncate(time.Minute)
		}
		win := 60
		keys := make([]string, 0, win)
		for i := 0; i < win; i++ {
			keys = append(keys, f.rediscache.Key("hot:video:1m:%s", asOf.Add(-time.Duration(i)*time.Minute).Format("200601021504")))
		}
		dest := f.rediscache.Key("hot:video:merge:1m:%s", asOf.Format("200601021504")) //快照key:同一个as_of内复用
		opCtx, cancel := context.WithTimeout(ctx, 80*time.Millisecond)
		defer cancel()
		existed, _ := f.rediscache.Exists(opCtx, dest)
		if !existed {
			_ = f.rediscache.ZUnionStore(opCtx, dest, keys, "SUM")
			_ = f.rediscache.Expire(opCtx, dest, 2*time.Minute) // 给翻页留时间
		}
		start := int64(offset)
		stop := start + int64(limit) - 1
		members, err := f.rediscache.ZRevRange(opCtx, dest, start, stop)
		if err == nil && len(members) == 0 {
			if offset > 0 {
				return ListByPopularityResponse{
					VideoList:  []FeedVideoItem{},
					AsOf:       reqAsof,
					NextOffset: offset,
					HasMore:    false,
				}, nil
			}
		}
		if err == nil && len(members) > 0 {
			ids := make([]uint, 0, len(members))
			for _, idStr := range members {
				id, err := strconv.ParseUint(idStr, 10, 64)
				if err == nil && id > 0 {
					ids = append(ids, uint(id))
				}
			}
			videos, err := f.repo.GetByIDs(ctx, ids)
			if err == nil {
				m := make(map[uint]*video.Video, len(videos))
				for _, v := range videos {
					m[v.ID] = v
				}
				ordered := make([]*video.Video, 0, len(ids))
				for _, id := range ids {
					if v := m[id]; v != nil {
						ordered = append(ordered, v)
					}
				}
				feedVideoItems, err := f.buildFeedVideos(ctx, ordered, viewerAccountID)
				if err != nil {
					return ListByPopularityResponse{}, err
				}
				resp := ListByPopularityResponse{
					VideoList:  feedVideoItems,
					AsOf:       reqAsof,
					NextOffset: offset + len(feedVideoItems),
					HasMore:    len(feedVideoItems) == limit,
				}
				if len(ordered) > 0 {
					last := ordered[len(ordered)-1]
					nextPopularity := last.Popularity
					nextBefore := last.CreateTime
					nextID := last.ID
					resp.NextLatestPopularity = &nextPopularity
					resp.NextLatestBefore = &nextBefore
					resp.NextLatestIDBefore = &nextID
				}
				return resp, nil
			}
		}
	}
	videos, err := f.repo.ListByPopularity(ctx, limit, latestPopuLarity, latestBefore, latestIDBefore)
	if err != nil {
		return ListByPopularityResponse{}, err
	}
	items, err := f.buildFeedVideos(ctx, videos, viewerAccountID)
	if err != nil {
		return ListByPopularityResponse{}, err
	}
	resp := ListByPopularityResponse{
		VideoList:  items,
		AsOf:       0,
		NextOffset: 0,
		HasMore:    len(items) == limit,
	}
	if len(videos) > 0 {
		last := videos[len(videos)-1]
		nextPopularity := last.Popularity
		nextBefore := last.CreateTime
		nextID := last.ID
		resp.NextLatestPopularity = &nextPopularity
		resp.NextLatestBefore = &nextBefore
		resp.NextLatestIDBefore = &nextID
	}
	return resp, nil
}

func (f *FeedService) buildFeedVideos(ctx context.Context, videos []*video.Video, viewerAccountID uint) ([]FeedVideoItem, error) {
	videoIDs := make([]uint, 0, len(videos))
	for _, v := range videos {
		videoIDs = append(videoIDs, v.ID)
	}
	likedMap, _ := f.repo.BatchGetLikedVideoIDs(ctx, viewerAccountID, videoIDs)

	feedVideos := make([]FeedVideoItem, 0, len(videos))
	for _, video := range videos {
		feedVideos = append(feedVideos, FeedVideoItem{
			ID:          video.ID,
			Author:      FeedAuthor{ID: video.AuthorID, Username: video.Username},
			Title:       video.Title,
			Description: video.Description,
			PlayURL:     video.PlayURL,
			CoverURL:    video.CoverURL,
			CreateTime:  video.CreateTime.Unix(),
			LikesCount:  video.LikesCount,
			IsLiked:     likedMap[video.ID],
		})
	}
	return feedVideos, nil
}

func buildOrderedResult(orderedIDs []uint, dataMap map[uint]*video.Video) []*video.Video {
	res := make([]*video.Video, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		if v, exists := dataMap[id]; exists && v != nil {
			res = append(res, v)
		}
	}
	return res
}

func (f *FeedService) ListByTag(ctx context.Context, tagName string, limit int, viewerAccountID uint) ([]FeedVideoItem, error) {
	videos, err := f.repo.ListByTag(ctx, tagName, limit)
	if err != nil {
		return nil, err
	}
	return f.buildFeedVideos(ctx, videos, viewerAccountID)
}
