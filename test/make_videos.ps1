# ===== Batch create 20 videos for Feed pagination test =====
# Usage: cd D:\feedsystem_video\test ; .\make_videos.ps1
$BASE = 'http://localhost:8080'

$login = Invoke-RestMethod -Uri "$BASE/account/login" -Method Post -ContentType 'application/json' -Body '{"username":"feedtest","password":"pass123456"}'
$token = $login.token
Write-Host "login OK, token: $($token.Substring(0,10))..."

for ($i = 1; $i -le 20; $i++) {
    $body = @{ title = " $i #"; description = 'seed data'; play_url = 'http://localhost:8080/static/videos/fake.mp4'; cover_url = 'http://localhost:8080/static/covers/fake.jpg' } | ConvertTo-Json
    $resp = Invoke-RestMethod -Uri "$BASE/video/publish" -Method Post `
        -Headers @{ Authorization = "Bearer $token" } `
        -ContentType 'application/json' -Body $body
    Write-Host " $i : id=$($resp.id)"
}
Write-Host 'done'
