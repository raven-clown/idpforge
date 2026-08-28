param(
    [string]$BinaryPath = "C:\Program Files\idpforge\idpforge-server.exe",
    [string]$ServiceName = "idpforge"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $BinaryPath)) {
    throw "Binary not found at $BinaryPath. Copy idpforge-server.exe there first."
}

$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "Service $ServiceName already exists, stopping and removing first."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

New-Service -Name $ServiceName `
    -BinaryPathName "`"$BinaryPath`"" `
    -DisplayName "IdpForge SSO Server" `
    -Description "Self-hosted SSO/identity platform" `
    -StartupType Automatic

Write-Host "Service $ServiceName installed. Configure environment variables in the registry"
Write-Host "under HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName\Environment, or wrap"
Write-Host "the binary with a launcher that loads C:\ProgramData\idpforge\idpforge.env."
Write-Host "Start it with: Start-Service $ServiceName"
