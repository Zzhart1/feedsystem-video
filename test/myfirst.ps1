$resp = Invoke-RestMethod -Uri 'http://localhost:8080/feed/listLatest' -Method Post -ContentType 'application/json' -Body '{"limit":3}'
foreach ($v in $resp.video_list) {
    Write-Host "id=$($v.id) 标题=$($v.title)"
}