# ===== Stage 3 Feed verification: pagination & ordering =====
# Usage: cd D:\feedsystem_video\test ; .\test_feed.ps1
$BASE = 'http://localhost:8080'

Write-Host '===== Test 1: listLatest 3 pages, check duplicates ====='
$allIDs = @()
$latestTime = 0
$hasMore = $true

for ($page = 1; $page -le 3 -and $hasMore; $page++) {
    $body = @{ limit = 10; latest_time = $latestTime } | ConvertTo-Json
    $resp = Invoke-RestMethod -Uri "$BASE/feed/listLatest" -Method Post -ContentType 'application/json' -Body $body
    $pageIDs = @()
    foreach ($v in $resp.video_list) {
        $allIDs += $v.id
        $pageIDs += $v.id
    }
    Write-Host "page $page : got $($resp.video_list.Count) items, next_time=$($resp.next_time), ids=[$($pageIDs -join ',')]"
    $latestTime = $resp.next_time
    $hasMore = $resp.has_more
}
$dup = $allIDs | Group-Object | Where-Object { $_.Count -gt 1 }
if ($dup) {
    Write-Host "FAIL: duplicates found: $($dup.Name -join ',')"
} else {
    Write-Host "PASS: $($allIDs.Count) unique ids across 3 pages, no duplicates"
}

Write-Host ''
Write-Host '===== Test 2: listLikesCount ordering ====='
$body2 = @{ limit = 10 } | ConvertTo-Json
$r2 = Invoke-RestMethod -Uri "$BASE/feed/listLikesCount" -Method Post -ContentType 'application/json' -Body $body2
$prev = [long]::MaxValue
$ok = $true
foreach ($v in $r2.video_list) {
    Write-Host "id=$($v.id) likes=$($v.likes_count) title=$($v.title)"
    if ($v.likes_count -gt $prev) { $ok = $false }
    $prev = $v.likes_count
}
if ($ok) { Write-Host 'PASS: likes_count is descending' } else { Write-Host 'FAIL: ordering broken' }
