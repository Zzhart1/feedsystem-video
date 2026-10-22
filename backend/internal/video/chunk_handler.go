package video

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"feedsystem_video_go/internal/middleware/jwt"
	rediscache "feedsystem_video_go/internal/middleware/redis"

	"github.com/gin-gonic/gin"
)

const sessionTTL = 24 * time.Hour

var errChunkCacheUnavailable = errors.New("chunk upload requires redis")

type ChunkUploadHandler struct {
	cache *rediscache.Client
}

func NewChunkUploadHandler(cache *rediscache.Client) *ChunkUploadHandler {
	return &ChunkUploadHandler{cache: cache}
}

func (h *ChunkUploadHandler) sessionKey(uploadID string) string {
	return h.cache.Key("chunk_upload:%s", uploadID)
}

func (h *ChunkUploadHandler) hashKey(accountID uint, fileHash string) string {
	return h.cache.Key("chunk_upload_hash:%d:%s", accountID, fileHash)
}

func (h *ChunkUploadHandler) getSession(c *gin.Context, uploadID string) (*ChunkUploadSession, error) {
	if h.cache == nil {
		return nil, errChunkCacheUnavailable
	}
	b, err := h.cache.GetBytes(c.Request.Context(), h.sessionKey(uploadID))
	if err != nil {
		return nil, err
	}
	var s ChunkUploadSession
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (h *ChunkUploadHandler) saveSession(c *gin.Context, s *ChunkUploadSession) error {
	if h.cache == nil {
		return errChunkCacheUnavailable
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return h.cache.SetBytes(c.Request.Context(), h.sessionKey(s.UploadID), b, sessionTTL)
}

func (h *ChunkUploadHandler) InitChunkUpload(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusInternalServerError, errChunkCacheUnavailable)
		return
	}
	var req InitChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}
	const maxSize = 200 << 20
	if req.FileSize > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filesize is big"})
		return
	}
	hashKey := h.hashKey(accountID, req.FileHash)
	existingID, err := h.cache.GetBytes(c.Request.Context(), hashKey)
	if err == nil && len(existingID) > 0 {
		session, err := h.getSession(c, string(existingID))
		if err == nil {
			_ = h.saveSession(c, session)
			h.cache.SetBytes(c.Request.Context(), hashKey, existingID, sessionTTL)
			c.JSON(200, gin.H{
				"upload_id":       session.UploadID,
				"uploaded_chunks": session.UploadedChunks(),
			})
			return
		}
	}
	id, _ := randHex(16)
	uploadID := id + fmt.Sprintf("%d", time.Now().UnixNano())
	session := &ChunkUploadSession{
		UploadID:     uploadID,
		AccountID:    accountID,
		Filename:     req.Filename,
		FileSize:     req.FileSize,
		FileHash:     req.FileHash,
		ChunkSize:    req.ChunkSize,
		TotalChunks:  req.TotalChunks,
		UploadedBits: make([]bool, req.TotalChunks),
	}
	if err := h.saveSession(c, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	if err := h.cache.SetBytes(c.Request.Context(), hashKey, []byte(uploadID), sessionTTL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	c.JSON(200, gin.H{
		"upload_id":       uploadID,
		"uploaded_chunks": []int{},
	})
}

func (h *ChunkUploadHandler) UploadChunk(c *gin.Context) {
	//cache是否存在
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable})
		return
	}
	//获取参数req
	var req UploadChunkRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	//获取session,获取用户id,校验id
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if req.ChunkIndex < 0 || req.ChunkIndex >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}
	if session.UploadedBits[req.ChunkIndex] {
		c.JSON(http.StatusOK, gin.H{"chunk_index": req.ChunkIndex})
		return
	}
	//md5校验
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errro": "missing file"})
		return
	}
	chunkfile, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}
	defer chunkfile.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, chunkfile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash chunk"})
		return
	}
	actualhash := fmt.Sprintf("%x", hash.Sum(nil))
	if actualhash != req.ChunkHash {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk hash mismatch", "expected": req.ChunkHash, "actual": actualhash})
		return
	}
	tmpdir := filepath.Join(".run", "uploads", "tmp", req.UploadID)
	if err := os.MkdirAll(tmpdir, 0o75); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}
	chunkpath := filepath.Join(tmpdir, fmt.Sprintf("%d", req.ChunkIndex))
	if _, err := chunkfile.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}
	dst, err := os.Create(chunkpath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, chunkfile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}
	session.UploadedBits[req.ChunkIndex] = true
	if err := h.saveSession(c, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"chunk_index": req.ChunkIndex})
}

func (h *ChunkUploadHandler) ChunkStatus(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return
	}
	var req ChunkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if accountID != session.AccountID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(200, gin.H{
		"upload_id":     session.UploadID,
		"uploadedchunk": session.UploadedChunks(),
		"total_chunks":  session.TotalChunks,
	})
}

func (h *ChunkUploadHandler) CompleteChunkUpload(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return
	}
	var req CompleteChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "forbidden"})
		return
	}
	if !session.IsComplete() {
		missing := 0
		for _, upload := range session.UploadedBits {
			if !upload {
				missing++
				if missing > 5 {
					missing = 5
					break
				}
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "not all chunks uploaded",
			"missing":      missing,
			"completed":    session.UploadedChunks(),
			"total_chunks": session.TotalChunks,
		})
	}
	//complete
	//创建文件夹
	date := time.Now().Format("20060102")
	reldir := filepath.Join("videos", fmt.Sprintf("%d", accountID), date)
	root := filepath.Join(".run", "uploads")
	absdir := filepath.Join(root, reldir)
	if err := os.MkdirAll(absdir, 0o75); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create output dir"})
		return
	}

	//上传文件
	filename, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	finalPath := filepath.Join(absdir, filename+".mp4")
	finalFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create final file"})
		return
	}
	defer finalFile.Close()

	tmpDir := filepath.Join(".run", "uploads", "tmp", req.UploadID)
	for i := 0; i < session.TotalChunks; i++ {
		chunkpath := filepath.Join(tmpDir, fmt.Sprintf("%d", i))
		cf, err := os.Open(chunkpath)
		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("chunk %d missing", i)})
			return
		}
		_, err = io.Copy(finalFile, cf)
		cf.Close()
		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge chunks"})
			return
		}
	}
	finalFile.Close()
	//clean up all tmp chunks
	_ = os.RemoveAll(tmpDir)

	//clean up redis session
	_ = h.cache.Del(c.Request.Context(), h.sessionKey(req.UploadID))
	_ = h.cache.Del(c.Request.Context(), h.hashKey(accountID, session.FileHash))

	urlPath := fmt.Sprintf("/static/videos/%d/%s/%s.mp4", accountID, date, filename)
	playURL := buildAbsoluteURL(c, urlPath)

	c.JSON(http.StatusOK, gin.H{
		"url":      playURL,
		"play_url": playURL,
	})
}
