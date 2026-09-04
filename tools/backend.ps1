# Starts, checks or stops the local backend (klondiked) for development.
#
#   pwsh -File tools/backend.ps1                 # ensure: reuse a healthy backend, else build and start
#   pwsh -File tools/backend.ps1 -Status         # report only
#   pwsh -File tools/backend.ps1 -Restart -Announce 2 -Delay 7s   # demo-world controls
#   pwsh -File tools/backend.ps1 -Stop
#
# "Healthy" means /v1/health answers on the loopback port with the contract version
# the source tree declares. A backend started by hand with the same values is left
# alone; one announcing something else is only replaced with -Restart.

param(
    [switch]$Status,
    [switch]$Restart,
    [switch]$Stop,
    [string]$Announce = '',
    [string]$Delay = '',
    [int]$Port = 8471
)

$ErrorActionPreference = 'Stop'
$backendDir = Join-Path $PSScriptRoot '..\backend' | Resolve-Path
$exe = Join-Path $backendDir 'klondiked.exe'
$dataDir = Join-Path $backendDir 'data'
$seed = Join-Path $dataDir 'klondiked.json'
$url = "http://127.0.0.1:$Port/v1/health"

$contract = (Select-String -Path (Join-Path $backendDir 'internal\domain\reason.go') -Pattern 'ContractVersion = "([^"]+)"').Matches[0].Groups[1].Value
$expected = if ($Announce) { $Announce } else { $contract }

# The -delay flag delays every answer, health included, so the check waits longer
# than the delay when one is set.
$healthTimeout = 3
if ($Delay -match '^(\d+(?:\.\d+)?)(ms|s|m)?$') {
    $seconds = [double]$Matches[1]
    switch ($Matches[2]) { 'ms' { $seconds /= 1000 } 'm' { $seconds *= 60 } }
    $healthTimeout = [int][math]::Ceiling($seconds) + 3
}

function Get-Health {
    try { return (Invoke-WebRequest -UseBasicParsing -TimeoutSec $healthTimeout $url).Content | ConvertFrom-Json } catch { return $null }
}

function Stop-Backend {
    $procs = Get-Process klondiked -ErrorAction SilentlyContinue
    if ($procs) { $procs | Stop-Process -Force; Start-Sleep -Milliseconds 500; Write-Host "stopped klondiked ($($procs.Count) process(es))" }
    else { Write-Host 'no klondiked process' }
}

$health = Get-Health
if ($Status) {
    if ($health) { Write-Host "backend: up on port $Port, contract $($health.contract) (source declares $contract), config revision $($health.config_revision), answer limit $($health.answer_timeout_s) s, re-check $($health.recheck_interval_s) s" }
    else { Write-Host "backend: nothing answers on port $Port" }
    exit 0
}

if ($Stop) { Stop-Backend; exit 0 }

if ($health -and -not $Restart) {
    if ($health.contract -eq $expected) { Write-Host "backend already up on port $Port, contract $($health.contract); nothing to do"; exit 0 }
    Write-Host "backend on port $Port announces contract $($health.contract), expected $expected; use -Restart to replace it"
    exit 1
}

if ($health -or (Get-Process klondiked -ErrorAction SilentlyContinue)) { Stop-Backend }

Push-Location $backendDir
try {
    go build -o klondiked.exe ./cmd/klondiked
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
} finally { Pop-Location }

New-Item -ItemType Directory -Force $dataDir | Out-Null
if (-not (Test-Path $seed)) {
    '{ "operators": [], "answer_timeout_s": 5, "recheck_interval_s": 15 }' | Set-Content -Encoding ascii $seed
    Write-Host "wrote $seed with an EMPTY operator list; add the operator's player identity (printed by the server log at connect) before testing cockpit commands"
}

$args = @('-listen', "127.0.0.1:$Port", '-data', 'data')
if ($Announce) { $args += @('-announce-contract', $Announce) }
if ($Delay) { $args += @('-delay', $Delay) }
$proc = Start-Process -FilePath $exe -ArgumentList $args -WorkingDirectory $backendDir -PassThru -WindowStyle Minimized

for ($i = 0; $i -lt 20; $i++) {
    Start-Sleep -Milliseconds 250
    $health = Get-Health
    if ($health) { break }
}
if (-not $health) { throw "backend started (pid $($proc.Id)) but /v1/health did not answer within $($healthTimeout + 5) s" }
Write-Host "backend started (pid $($proc.Id)) on port $Port, contract $($health.contract), config revision $($health.config_revision)$(if ($Delay) { ", delay $Delay" })"
$seedJson = Get-Content $seed -Raw | ConvertFrom-Json
if (-not $seedJson.operators -or $seedJson.operators.Count -eq 0) { Write-Host 'note: the seed has no operators; cockpit commands will be refused as unauthorized' }
