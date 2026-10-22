<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import SparkMD5 from 'spark-md5'
import { uploadVideo, uploadCover, publishVideo, initChunkUpload, uploadChunk, chunkStatus, completeChunkUpload } from '../api/video'
import Toaster from '../utils/toast'

const router = useRouter()
const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB，和后端 ChunkSize 一致

const title = ref('')
const description = ref('')
const tags = ref('')
const videoFile = ref(null)
const coverFile = ref(null)
const coverUrl = ref('')
const videoPreview = ref('')

const uploading = ref(false)
const uploadMode = ref('') // '' | 'direct' | 'chunk'
const progress = ref(0) // 0-100
const progressText = ref('')

function onPickVideo(e) {
  const f = e.target.files?.[0]
  if (!f) return
  videoFile.value = f
  videoPreview.value = URL.createObjectURL(f)
  progress.value = 0
  progressText.value = ''
}

function onPickCover(e) {
  const f = e.target.files?.[0]
  if (!f) return
  coverFile.value = f
  coverUrl.value = URL.createObjectURL(f)
}

// 整体文件 MD5（spark-md5 增量计算，避免大文件卡 UI）
function calcFileHash(file) {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    const chunk = 4 * 1024 * 1024
    let pos = 0
    reader.onerror = () => reject(new Error('读取文件失败'))
    reader.onload = (e) => {
      spark.append(e.target.result)
      pos += chunk
      if (pos < file.size) {
        reader.readAsArrayBuffer(file.slice(pos, pos + chunk))
      } else {
        resolve(spark.end())
      }
    }
    reader.readAsArrayBuffer(file.slice(0, chunk))
  })
}

function calcBlobHash(blob) {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    reader.onerror = () => reject(new Error('读取分片失败'))
    reader.onload = (e) => {
      spark.append(e.target.result)
      resolve(spark.end())
    }
    reader.readAsArrayBuffer(blob)
  })
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }

async function startUpload() {
  if (!videoFile.value) {
    Toaster.error('请先选择视频文件')
    return
  }
  if (!title.value.trim()) {
    Toaster.error('请填写标题')
    return
  }
  uploading.value = true
  try {
    const file = videoFile.value
    // 小文件走普通上传，大文件走分片（可断点续传）
    if (file.size < CHUNK_SIZE) {
      uploadMode.value = 'direct'
      progressText.value = '普通上传中…'
      const res = await uploadVideo(file)
      progress.value = 100
      await doPublish(res.url || res.play_url)
      return
    }

    uploadMode.value = 'chunk'
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
    progressText.value = `计算文件 MD5（共 ${totalChunks} 片）…`
    const fileHash = await calcFileHash(file)

    progressText.value = '初始化分片会话…'
    const initRes = await initChunkUpload({
      file_name: file.name,
      file_size: file.size,
      chunk_size: CHUNK_SIZE,
      total_chunks: totalChunks,
      file_hash: fileHash,
    })
    const uploadId = initRes.upload_id

    // 断点续传：查询服务端已收到的片，跳过
    let uploaded = []
    try {
      const st = await chunkStatus(uploadId)
      uploaded = st.uploaded_chunks || []
    } catch { /* 查询失败则从头传 */ }

    for (let i = 0; i < totalChunks; i++) {
      if (uploaded.includes(i)) continue
      const start = i * CHUNK_SIZE
      const blob = file.slice(start, start + CHUNK_SIZE)
      const chunkHash = await calcBlobHash(blob)
      let ok = false
      for (let attempt = 0; attempt < 3 && !ok; attempt++) {
        try {
          await uploadChunk(uploadId, i, chunkHash, blob)
          ok = true
        } catch (e) {
          if (attempt === 2) throw e
          await sleep(800)
        }
      }
      progress.value = Math.round(((i + 1) / totalChunks) * 100)
      progressText.value = `上传分片 ${i + 1}/${totalChunks}`
    }

    progressText.value = '合并文件…'
    const done = await completeChunkUpload(uploadId)
    progress.value = 100
    await doPublish(done.url || done.play_url)
  } catch (e) {
    Toaster.error(e.message || '上传失败（已上传的分片会自动续传，重新点击发布即可）')
  } finally {
    uploading.value = false
  }
}

async function doPublish(playUrl) {
  // 上传封面（可选）
  let cover = ''
  if (coverFile.value) {
    progressText.value = '上传封面…'
    try {
      const c = await uploadCover(coverFile.value)
      cover = c.url || c.cover_url || ''
    } catch { /* 封面失败不阻塞发布 */ }
  }
  progressText.value = '发布中…'
  const tagList = tags.value.split(/[,#，\s]+/).map((t) => t.trim()).filter(Boolean)
  await publishVideo({ title: title.value, description: description.value, play_url: playUrl, cover_url: cover, tags: tagList.join(',') })
  Toaster.success('发布成功！')
  router.push('/')
}
</script>

<template>
  <div class="page">
    <div class="card box">
      <h1>发布视频</h1>
      <p class="sub">小于 5MB 走普通上传；大于 5MB 自动分片（支持断点续传，中断后重试会跳过已上传的分片）</p>

      <div class="fields">
        <label>视频文件</label>
        <div class="drop" :class="{ has: videoFile }" @click="$refs.videoInput.click()" @dragover.prevent @drop.prevent="onPickVideo({ target: { files: $event.dataTransfer.files } })">
          <input ref="videoInput" type="file" accept="video/*" hidden @change="onPickVideo" />
          <template v-if="!videoFile">
            <div class="ic">🎬</div>
            <div>点击或拖拽视频到这里</div>
            <div class="tip">支持 mp4 / webm 等格式</div>
          </template>
          <template v-else>
            <video v-if="videoPreview" :src="videoPreview" class="prev" controls muted></video>
            <div class="fname">{{ videoFile.name }} · {{ (videoFile.size / 1024 / 1024).toFixed(1) }}MB</div>
          </template>
        </div>

        <label>封面图（可选）</label>
        <div class="drop cover" @click="$refs.coverInput.click()">
          <input ref="coverInput" type="file" accept="image/*" hidden @change="onPickCover" />
          <img v-if="coverUrl" :src="coverUrl" class="cover-prev" alt="" />
          <div v-else class="tip">点击选择封面（jpg/png/webp）</div>
        </div>

        <label>标题</label>
        <input v-model="title" class="input" maxlength="80" placeholder="起个吸引人的标题" />

        <label>描述</label>
        <textarea v-model="description" class="input ta" rows="3" placeholder="介绍一下这个视频…"></textarea>

        <label>标签（可选，用逗号或空格分隔）</label>
        <input v-model="tags" class="input" placeholder="例如：Go 后端 短视频" />
      </div>

      <div v-if="uploading" class="uploading">
        <div class="bar"><div class="fill" :style="{ width: progress + '%' }"></div></div>
        <div class="ptext">{{ progressText }} · {{ progress }}%</div>
      </div>

      <button class="btn pub" :disabled="uploading" @click="startUpload">
        {{ uploading ? '上传中…' : '发布视频' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.box { max-width: 640px; margin: 0 auto; padding: 28px 30px 30px; }
h1 { font-size: 20px; margin-bottom: 4px; }
.sub { color: var(--text-2); font-size: 13px; margin-bottom: 20px; }
.fields label { display: block; font-size: 13px; font-weight: 600; color: var(--text-2); margin: 14px 0 6px; }
.drop {
  border: 2px dashed var(--border); border-radius: var(--radius);
  padding: 22px; text-align: center; cursor: pointer;
  transition: border-color 0.14s ease; background: var(--surface);
}
.drop:hover { border-color: var(--accent); }
.drop .ic { font-size: 34px; }
.drop .tip { color: var(--text-2); font-size: 12px; margin-top: 4px; }
.prev { width: 100%; max-height: 220px; border-radius: 8px; background: #000; }
.fname { margin-top: 8px; font-weight: 600; font-size: 13px; }
.cover { padding: 12px; }
.cover-prev { width: 100%; max-height: 180px; object-fit: cover; border-radius: 8px; }
.ta { resize: vertical; }
.uploading { margin: 18px 0 6px; }
.bar { height: 10px; border-radius: 999px; background: #eef2f6; overflow: hidden; }
.fill { height: 100%; background: linear-gradient(90deg, #0ea5e9, #22c55e); transition: width 0.2s ease; }
.ptext { font-size: 12.5px; color: var(--text-2); margin-top: 6px; text-align: center; }
.pub { width: 100%; padding: 12px; margin-top: 14px; font-size: 15px; }
</style>
