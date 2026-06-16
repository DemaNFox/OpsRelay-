param(
  [string]$Output = "dist\opsrelay-agent-windows-amd64.exe",
  [switch]$Console
)

$ErrorActionPreference = "Stop"

Push-Location (Join-Path $PSScriptRoot "..")
try {
  New-Item -ItemType Directory -Force -Path (Split-Path $Output) | Out-Null
  $env:GOOS = "windows"
  $env:GOARCH = "amd64"
  if (Get-Command rsrc -ErrorAction SilentlyContinue) {
    rsrc -manifest "cmd\opsrelay-agent\opsrelay-agent.exe.manifest" -o "cmd\opsrelay-agent\rsrc_windows_amd64.syso"
  } else {
    Write-Warning "rsrc was not found; GUI builds may fail to start without embedded Common Controls manifest."
  }
  $ldflags = "-s -w"
  if (-not $Console) {
    $ldflags = "$ldflags -H=windowsgui"
  }
  go build -trimpath "-ldflags=$ldflags" -o $Output ./cmd/opsrelay-agent
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
  }
  Write-Output "Built $Output"
}
finally {
  Pop-Location
}
