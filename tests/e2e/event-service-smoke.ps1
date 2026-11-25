param(
    [string]$AccountApi = 'http://localhost:8081/api/v1',
    [string]$EventApi = 'http://localhost:8082/api/v1',
    [string]$AdminEmail = 'admin@example.test',
    [string]$AdminPassword = 'MatchMateDev123!'
)

function Assert-ApiReady {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$HealthUrl,
        [Parameter(Mandatory = $true)][string]$StartHint
    )

    try {
        $health = Invoke-RestMethod -Method Get -Uri $HealthUrl -TimeoutSec 5 -ErrorAction Stop
        if ($health.status -notin @('ok', 'ready')) {
            throw "Unexpected health status: $($health.status)"
        }
    }
    catch {
        throw "$Name is not available at $HealthUrl. $StartHint Original error: $($_.Exception.Message)"
    }
}

Assert-ApiReady -Name 'Account API' -HealthUrl ($AccountApi -replace '/api/v1$', '/health/ready') -StartHint 'Start Account PostgreSQL/migrations and run Account API on port 8081.'
Assert-ApiReady -Name 'Event API' -HealthUrl ($EventApi -replace '/api/v1$', '/health/ready') -StartHint 'Start the Event Compose api profile on port 8082.'

$loginBody = @{ email = $AdminEmail; password = $AdminPassword } | ConvertTo-Json
$login = Invoke-RestMethod -Method Post -Uri "$AccountApi/auth/login" -ContentType 'application/json' -Body $loginBody -ErrorAction Stop
$authHeaders = @{ Authorization = "Bearer $($login.accessToken)"; 'X-Correlation-ID' = "event-smoke-$([guid]::NewGuid())" }
$me = Invoke-RestMethod -Method Get -Uri "$AccountApi/users/me" -Headers $authHeaders -ErrorAction Stop
if ($me.account.roles -notcontains 'admin') { throw 'Development account does not have administrator authority.' }

$start = [DateTime]::UtcNow.AddDays(14)
$createBody = @{
    organizerId = $me.account.id
    name = 'Fictional Event Service Smoke Social'
    description = 'Development-only event created by the automated smoke test.'
    venueName = 'Private Test Venue'
    broadLocation = 'Colombo'
    timeZone = 'Asia/Colombo'
    startsAt = $start.ToString('o')
    endsAt = $start.AddHours(3).ToString('o')
    registrationOpensAt = $start.AddDays(-10).ToString('o')
    registrationClosesAt = $start.AddDays(-1).ToString('o')
    price = '4500.00'
    currency = 'LKR'
    configuredCapacity = 60
    matchingRulesetVersion = 'rules-v1'
} | ConvertTo-Json

$created = Invoke-RestMethod -Method Post -Uri "$EventApi/events" -Headers $authHeaders -ContentType 'application/json' -Body $createBody -ErrorAction Stop
$publishBody = @{ expectedVersion = $created.version; reason = '' } | ConvertTo-Json
$published = Invoke-RestMethod -Method Post -Uri "$EventApi/events/$($created.eventId)/publish" -Headers $authHeaders -ContentType 'application/json' -Body $publishBody -ErrorAction Stop
$public = Invoke-RestMethod -Method Get -Uri "$EventApi/events/$($created.eventId)" -ErrorAction Stop

if ($published.status -ne 'PUBLISHED') { throw 'Event did not enter PUBLISHED state.' }
if ($null -ne $public.organizerId -or $null -ne $public.venueName) { throw 'Public event response leaked operational fields.' }
Write-Output "Event smoke test passed for event $($created.eventId)."
