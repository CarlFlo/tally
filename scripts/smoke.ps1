param([string]$BaseURL = 'http://127.0.0.1:18081')
$ErrorActionPreference = 'Stop'
# Run only against a fresh, disposable test deployment. This imports a real show.
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$headers = @{ 'X-Tally-CSRF' = '1' }
function Call-App([string]$Method, [string]$Path, $Body = $null) {
    $args = @{ Uri = "$BaseURL/api$Path"; Method = $Method; WebSession = $session; Headers = $headers; TimeoutSec = 110 }
    if ($null -ne $Body) { $args.ContentType = 'application/json'; $args.Body = ConvertTo-Json -InputObject $Body -Compress }
    Invoke-RestMethod @args
}
function Assert-That([bool]$Condition, [string]$Message) { if (!$Condition) { throw $Message } }
$ready = Invoke-RestMethod -Uri "$BaseURL/readyz" -TimeoutSec 5
Assert-That ($ready.status -eq 'ready') 'Container is not ready'
$boot = Call-App GET '/bootstrap'
Assert-That ($boot.profile.id -eq 'user0' -and $boot.profiles.Count -eq 1) 'Smoke test requires a fresh single-profile deployment'
$results = Call-App GET '/shows/search?q=Severance'
$candidate = $results | Where-Object { $_.show.name -eq 'Severance' } | Select-Object -First 1
Assert-That ($null -ne $candidate) 'TVmaze ranked search did not return the requested title'
$added = Call-App POST '/shows' @{ tvmaze_id = $candidate.show.id }
$detail = Call-App GET "/shows/$($added.id)"
Assert-That ($detail.episodes.Count -gt 0 -and $detail.seasons.Count -gt 0) 'Episodes or seasons were not imported'
$episode = $detail.episodes | Where-Object { $_.airdate -ne '' } | Select-Object -First 1
Call-App PATCH "/episodes/$($episode.id)" @{ watched = $true } | Out-Null
$statsBefore = Call-App GET '/statistics'
$callsBefore = ($statsBefore.summary | Where-Object provider -eq 'tvmaze').requests
$from = ([DateTime]::Parse($episode.airdate)).ToString('yyyy-MM-01')
$to = ([DateTime]::Parse($from)).AddMonths(1).ToString('yyyy-MM-dd')
$calendar = Call-App GET "/calendar?from=$from&to=$to"
Assert-That ($calendar.Count -gt 0) 'Local calendar was empty after import'
Call-App POST '/profiles' @{ name = 'Smoke profile'; avatar = 'mint' } | Out-Null
Call-App POST '/auth/logout' @{} | Out-Null
Call-App POST '/profiles/select' @{ profile = 'user1' } | Out-Null
$shared = Call-App POST '/shows' @{ tvmaze_id = $candidate.show.id }
Assert-That ($shared.id -eq $added.id) 'Show metadata was not shared'
$second = Call-App GET "/shows/$($shared.id)"
Assert-That (($second.episodes | Where-Object id -eq $episode.id).watched -eq 0) 'Watched state leaked across profiles'
$statsAfter = Call-App GET '/statistics'
$callsAfter = ($statsAfter.summary | Where-Object provider -eq 'tvmaze').requests
Assert-That ($callsAfter -eq $callsBefore) 'Local calendar or second follow made a metadata API call'
$backup = Call-App POST '/jobs/backup' @{}
$finished = $false
for ($attempt = 0; $attempt -lt 40; $attempt++) {
    $jobs = Call-App GET '/jobs'
    $run = $jobs.runs | Where-Object id -eq $backup.id
    if ($run.status -eq 'success') { $finished = $true; break }
    if ($run.status -in @('failed', 'cancelled')) { throw "Backup failed: $($run.error)" }
    Start-Sleep -Milliseconds 250
}
Assert-That $finished 'Backup did not complete'
$settings = Call-App GET '/settings'
Assert-That ($settings.backups.Count -eq 1 -and $settings.backups[0].verified -eq 1) 'Verified backup record missing'
[PSCustomObject]@{ Result = 'PASS'; Show = $detail.show.name; Episodes = $detail.episodes.Count; Seasons = $detail.seasons.Count; TVmazeCalls = $callsAfter; SharedMetadata = $true; IsolatedState = $true; Backup = $settings.backups[0].filename } | ConvertTo-Json
