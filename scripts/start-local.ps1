Param(
  [int]$Port = 8080,
  [string]$RecoverPanics = "true",
  [string]$BuildTags = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:GOCACHE = Join-Path $root ".cache\go-build"
$env:GOPATH = Join-Path $root ".cache\gopath"
$env:GOMODCACHE = Join-Path $env:GOPATH "pkg\mod"
New-Item -ItemType Directory -Force -Path $env:GOCACHE, $env:GOMODCACHE | Out-Null

$localGo = Join-Path $root ".tools\go\bin\go.exe"
$goCmd = if (Test-Path $localGo) { $localGo } else { "go" }

$env:PORT = "$Port"
$env:BUGBOY_RECOVER_PANICS = $RecoverPanics

Write-Host "Starting Bugboy Go on http://localhost:$Port/ (BUGBOY_RECOVER_PANICS=$RecoverPanics)"
if ([string]::IsNullOrWhiteSpace($BuildTags)) {
  & $goCmd run ./cmd/server
} else {
  & $goCmd run -tags $BuildTags ./cmd/server
}
