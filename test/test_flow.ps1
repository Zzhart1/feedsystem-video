# ============================================================
# Phase 2 Verify A: normal upload flow
# register -> login -> upload cover/video -> publish(#tag) -> list -> detail x2
# usage: cd D:\feedsystem_video\test ; .\test_flow.ps1
# ============================================================
$ErrorActionPreference = 'Stop'
$BASE = 'http://localhost:8080'
$User = 'tester' + (Get-Random -Maximum 99999)
$Pass = 'pass123456'
$Mp4 = Join-Path $PSScriptRoot 'test.mp4'
$Jpg = Join-Path $PSScriptRoot 'test.jpg'

Write-Host '== 1. register ==' -ForegroundColor Cyan
try {
    $r = Invoke-RestMethod -Uri "$BASE/account/register" -Method Post -ContentType 'application/json' `
        -Body ('{"username":"' + $User + '","password":"' + $Pass + '"}')
    "register ok: $($r.message)"
} catch {
    "register skipped (exists or limited): $($_.Exception.Message)"
}

Write-Host '== 2. login ==' -ForegroundColor Cyan
$login = Invoke-RestMethod -Uri "$BASE/account/login" -Method Post -ContentType 'application/json' `
    -Body ('{"username":"' + $User + '","password":"' + $Pass + '"}')
$token = $login.token
"token head: $($token.Substring(0, [Math]::Min(20, $token.Length)))..."

Write-Host '== 3. upload cover ==' -ForegroundColor Cyan
$cover = (curl.exe -s -X POST "$BASE/video/uploadCover" -H "Authorization: Bearer $token" -F "file=@$Jpg" | Out-String)
$coverUrl = ($cover | ConvertFrom-Json).cover_url
"cover_url: $coverUrl"

Write-Host '== 4. upload video ==' -ForegroundColor Cyan
$up = (curl.exe -s -X POST "$BASE/video/uploadVideo" -H "Authorization: Bearer $token" -F "file=@$Mp4" | Out-String)
$playUrl = ($up | ConvertFrom-Json).play_url
"play_url: $playUrl"

Write-Host '== 5. publish (with #tag) ==' -ForegroundColor Cyan
$body = @{ title='My first video #phase2'; description='rebuild test'; play_url=$playUrl; cover_url=$coverUrl } | ConvertTo-Json -Compress
$pub = Invoke-RestMethod -Uri "$BASE/video/publish" -Method Post -Headers @{Authorization="Bearer $token"} -ContentType 'application/json' -Body $body
"publish ok, video id: $($pub.id)"

Write-Host '== 6. list by author ==' -ForegroundColor Cyan
$list = Invoke-RestMethod -Uri "$BASE/video/listByAuthorID" -Method Post -ContentType 'application/json' `
    -Body ('{"author_id":' + $pub.author_id + '}')
"found $($list.Count) videos"

Write-Host '== 7. detail x2 ==' -ForegroundColor Cyan
foreach ($i in 1..2) {
    $d = Invoke-RestMethod -Uri "$BASE/video/getDetail" -Method Post -ContentType 'application/json' `
        -Body ('{"id":' + $pub.id + '}')
    "call $i : id=$($d.id) title=$($d.title)"
}
Write-Host '== A done ==' -ForegroundColor Green
