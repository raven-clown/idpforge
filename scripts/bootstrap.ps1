param()

$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

$ConfigDir = if ($env:IDPFORGE_CONFIG_DIR) { $env:IDPFORGE_CONFIG_DIR } else { "$env:ProgramData\idpforge" }
$DataDir   = if ($env:IDPFORGE_DATA_DIR)   { $env:IDPFORGE_DATA_DIR }   else { "$env:ProgramData\idpforge\data" }
$LogDir    = if ($env:IDPFORGE_LOG_DIR)    { $env:IDPFORGE_LOG_DIR }    else { "$env:ProgramData\idpforge\logs" }

foreach ($dir in @($ConfigDir, $DataDir, $LogDir)) {
    if (-not (Test-Path $dir)) {
        Write-Host "Creating $dir"
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }
}

$EnvFile = Join-Path $ConfigDir "idpforge.env"
if (-not (Test-Path $EnvFile)) {
    Write-Host "Writing $EnvFile from .env.example"
    Copy-Item (Join-Path $RootDir ".env.example") $EnvFile
    Write-Host "Edit $EnvFile before starting the service (set IDPFORGE_DB_DSN at minimum)."
} else {
    Write-Host "$EnvFile already exists, leaving it as is."
}

Write-Host "Bootstrap complete."
Write-Host "Next steps:"
Write-Host "  1. Edit $EnvFile"
Write-Host "  2. Copy the built binary to C:\Program Files\idpforge\idpforge-server.exe"
Write-Host "  3. Run deploy\windows\install-service.ps1"
Write-Host "  4. Start-Service idpforge"
