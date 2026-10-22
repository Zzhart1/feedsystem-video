# ============================================================
#  阶段四验收脚本：点赞 / 评论 / 关注 + 异步落库
#  （带语法教学注释，看完就会写 PowerShell 脚本）
#  运行前：API 和 Worker 都要开着：
#    终端1: cd D:\feedsystem_video\backend ; go run ./cmd
#    终端2: cd D:\feedsystem_video\backend ; go run ./cmd/worker
#  运行：  powershell -ExecutionPolicy Bypass -File D:\feedsystem_video\test\test_phase4.ps1
# ============================================================

# ---------- 语法速查（对照 C++，先看这五行）----------
# $变量 = 值                  → 变量（C++: auto x = ...）
# @{ key = val }              → 字典（C++: map<string,any>）
# 对象 | ConvertTo-Json       → 管道：把左边结果转成 JSON 字符串
# Invoke-RestMethod -Uri .. -Method POST -Body ..   → 发 HTTP 请求
#                              → 返回的 JSON 已自动解析成对象，用 $resp.字段 取值
# ------------------------------------------------

$base = "http://localhost:8080"   # ← 你的 API 地址

# ========== 0. 取一个测试视频（最新 Feed 的第一条）==========
# $listBody 是请求体：字典 @{} 用管道转成 JSON 字符串
$listBody = @{ limit = 5 } | ConvertTo-Json
# 发 POST 请求。listLatest 是软鉴权，不用带 token
$feed = Invoke-RestMethod -Uri "$base/feed/listLatest" -Method Post -Body $listBody -ContentType "application/json"
# 返回对象里取字段：.video_list 是视频数组，[0] 取第一个
$video = $feed.video_list[0]
$videoID = $video.id
$authorID = $video.author.id
$authorName = $video.author.username
Write-Host "测试视频 id=$videoID 作者=$authorName(id=$authorID) 当前点赞数=$($video.likes_count)"

# ========== 1. 注册 + 登录 用户B ==========
$userB = "phase4_user_b"
$pass = "123456"
$regBody = @{ username = $userB; password = $pass } | ConvertTo-Json
try {
    # try/catch：HTTP 报 4xx/5xx 时 Invoke-RestMethod 会抛异常。
    # 注册重名会报错，捕获后忽略，继续登录。
    Invoke-RestMethod -Uri "$base/account/register" -Method Post -Body $regBody -ContentType "application/json" | Out-Null
} catch { Write-Host "注册跳过（用户已存在）: $($_.Exception.Message)" }

$loginBody = @{ username = $userB; password = $pass } | ConvertTo-Json
$login = Invoke-RestMethod -Uri "$base/account/login" -Method Post -Body $loginBody -ContentType "application/json"
$token = $login.access_token                    # ← 如果报字段错，改成 $login.token 试试
$headers = @{ Authorization = "Bearer $token" } # JWT 认证头
Write-Host "用户B登录成功"

# ========== 2. 点赞 + 异步落库验证 ==========
$likeBody = @{ video_id = $videoID } | ConvertTo-Json
Invoke-RestMethod -Uri "$base/like/like" -Method Post -Headers $headers -Body $likeBody -ContentType "application/json" | Out-Null
Write-Host "已点赞。等 3 秒让 Worker 异步落库..."
Start-Sleep -Seconds 3

# 带 token 重新查 Feed（登录后 is_liked 才会返回）
$feed2 = Invoke-RestMethod -Uri "$base/feed/listLatest" -Method Post -Headers $headers -Body $listBody -ContentType "application/json"
# Where-Object 过滤数组：找出 id 等于目标视频的那一条（类似 C++ 的 std::find_if）
$v2 = $feed2.video_list | Where-Object { $_.id -eq $videoID }
if ($v2.is_liked -eq $true) { Write-Host "PASS 点赞态生效 is_liked=true" } else { Write-Host "FAIL is_liked=$($v2.is_liked)" }
Write-Host "点赞数变化: $($video.likes_count) -> $($v2.likes_count)"

# ========== 3. 评论 + 查列表 ==========
$commentBody = @{ video_id = $videoID; content = "阶段四测试评论 @$authorName" } | ConvertTo-Json
Invoke-RestMethod -Uri "$base/comment/publish" -Method Post -Headers $headers -Body $commentBody -ContentType "application/json" | Out-Null
Start-Sleep -Seconds 3
# 评论列表接口不确定返回结构，先原样打印看字段名
$comments = Invoke-RestMethod -Uri "$base/comment/listAll" -Method Post -Body @{ video_id = $videoID } | ConvertTo-Json -Compress -Depth 5
Write-Host "评论列表原始返回: $comments"

# ========== 4. 关注作者 + 关注流 ==========
$followBody = @{ vlogger_id = $authorID } | ConvertTo-Json
Invoke-RestMethod -Uri "$base/social/follow" -Method Post -Headers $headers -Body $followBody -ContentType "application/json" | Out-Null
$counts = Invoke-RestMethod -Uri "$base/social/getCounts" -Method Post -Headers $headers -Body $followBody -ContentType "application/json"
Write-Host "我的关注数 vlogger_count=$($counts.vlogger_count)"

# 关注流：B 关注了作者，应该能看到作者发的视频
$following = Invoke-RestMethod -Uri "$base/feed/listByFollowing" -Method Post -Headers $headers -Body $listBody -ContentType "application/json"
$found = $following.video_list | Where-Object { $_.id -eq $videoID }
if ($found) { Write-Host "PASS 关注流能看到视频 id=$videoID" } else { Write-Host "FAIL 关注流没看到该视频" }

Write-Host ""
Write-Host "===== 基础验收跑完 ====="
Write-Host "下一步自己动手（练翻页）：把上面的 limit 从 5 改成 2，跑两次 listByFollowing，"
Write-Host "对比两次返回的 id 有没有重复 —— 这就是游标翻页的'不重不漏'验证。"
