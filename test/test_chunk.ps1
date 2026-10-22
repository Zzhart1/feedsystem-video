# ============================================================
# Phase 2 Verify B: chunk upload + resume
# init -> upload first half -> status(resume check) -> rest -> complete
# usage: cd D:\feedsystem_video\test ; .\test_chunk.ps1
# ============================================================
$ErrorActionPreference = 'Stop'
$BASE = 'http://localhost:8080'
$User = 'chunker' + (Get-Random -Maximum 99999)
$Pass = 'pass123456'
$Mp4 = Join-Path $PSScriptRoot 'test.mp4'
$ChunkSize = 5MB
$TmpDir = Join-Path $PSScriptRoot 'chunks'

Write-Host '== register + login ==' -ForegroundColor Cyan
try {
    Invoke-RestMethod -Uri "$BASE/account/register" -Method Post -ContentType 'application/json' `
        -Body ('{"username":"' + $User + '","password":"' + $Pass + '"}') | Out-Null
} catch { }
$login = Invoke-RestMethod -Uri "$BASE/account/login" -Method Post -ContentType 'application/json' `
    -Body ('{"username":"' + $User + '","password":"' + $Pass + '"}')
$token = $login.token

$fi = Get-Item $Mp4
$fileHash = (Get-FileHash -Algorithm MD5 $Mp4).Hash.ToLower()
$total = [int][Math]::Ceiling($fi.Length / $ChunkSize)
"file: $($fi.Name)  size: $($fi.Length)  chunks: $total  md5: $fileHash"

Write-Host '== 1. chunk/init ==' -ForegroundColor Cyan
$body = @{ file_name=$fi.Name; chunk_size=$ChunkSize; file_size=$fi.Length; total_chunks=$total; file_hash=$fileHash } | ConvertTo-Json -Compress
$init = Invoke-RestMethod -Uri "$BASE/video/chunk/init" -Method Post -Headers @{Authorization="Bearer $token"} -ContentType 'application/json' -Body $body
$uploadId = $init.upload_id
"upload_id: $uploadId"

New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null
$md5 = [System.Security.Cryptography.MD5]::Create()
$fs = [System.IO.File]::OpenRead($Mp4)
$buf = New-Object byte[] $ChunkSize
$hashList = @{}
$half = [int][Math]::Floor($total / 2)

function Send-Chunk([int]$idx, [string]$tok, [string]$chunkHash) {
    $chunkPath = Join-Path $TmpDir ("chunk_$idx.bin")
    $resp = curl.exe -s -X POST "$BASE/video/chunk/upload" `
        -H "Authorization: Bearer $tok" `
        -F "upload_id=$uploadId" `
        -F "chunk_index=$idx" `
        -F "chunk_hash=$chunkHash" `
        -F "file=@$chunkPath" | Out-String
    "chunk $idx : $resp"
}

for ($i = 0; $i -lt $total; $i++) {
    $n = $fs.Read($buf, 0, $ChunkSize)
    $h = ($md5.ComputeHash($buf, 0, $n) | ForEach-Object { $_.ToString('x2') }) -join ''
    $hashList[$i] = $h
    $chunkPath = Join-Path $TmpDir ("chunk_$i.bin")
    [System.IO.File]::WriteAllBytes($chunkPath, $buf[0..($n - 1)])
    if ($i -lt $half) {
        Send-Chunk $i $token $h
    }
}
$fs.Close()

Write-Host "== 2. uploaded first $half chunks, check status ==" -ForegroundColor Cyan
$st = Invoke-RestMethod -Uri "$BASE/video/chunk/status" -Method Post -Headers @{Authorization="Bearer $token"} -ContentType 'application/json' -Body ('{"upload_id":"' + $uploadId + '"}')
"uploaded chunks: $($st.uploadedchunk -join ',')  total: $($st.total_chunks)"

Write-Host '== 3. upload the rest ==' -ForegroundColor Cyan
for ($i = $half; $i -lt $total; $i++) {
    Send-Chunk $i $token $hashList[$i]
}

Write-Host '== 4. chunk/complete ==' -ForegroundColor Cyan
$done = Invoke-RestMethod -Uri "$BASE/video/chunk/complete" -Method Post -Headers @{Authorization="Bearer $token"} -ContentType 'application/json' -Body ('{"upload_id":"' + $uploadId + '"}')
"play_url: $($done.play_url)"

Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
Write-Host '== B done ==' -ForegroundColor Green
