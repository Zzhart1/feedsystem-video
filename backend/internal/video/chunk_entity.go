package video

const ChunkSize = 5 << 20

type ChunkUploadSession struct {
	UploadID     string `json:"upload_id"`
	AccountID    uint   `json:"account_id"`
	TotalChunks  int    `json:"total_chunks"`
	ChunkSize    int64  `json:"chunk_size"`
	Filename     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	FileHash     string `json:"file_hash"`
	UploadedBits []bool `json:"uploaded_bits"`
}

func (s *ChunkUploadSession) UploadedChunks() []int {
	var indics []int
	for i, b := range s.UploadedBits {
		if b {
			indics = append(indics, i)
		}
	}
	return indics
}

func (s *ChunkUploadSession) IsComplete() bool {
	for _, b := range s.UploadedBits {
		if !b {
			return false
		}
	}
	return true
}

type InitChunkUploadRequest struct {
	TotalChunks int    `json:"total_chunks" binding:"required,min=1"`
	ChunkSize   int64  `json:"chunk_size" binding:"required,min=1"`
	Filename    string `json:"file_name" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required,min=1"`
	FileHash    string `json:"file_hash" binding:"required"`
}

type UploadChunkRequest struct {
	UploadID   string `form:"upload_id" binding:"required"`
	ChunkIndex int    `form:"chunk_index" binding:"min=0"`
	ChunkHash  string `form:"chunk_hash" binding:"required"`
}

type ChunkStatusRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

type CompleteChunkUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}
