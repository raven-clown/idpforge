param()

$ErrorActionPreference = "Stop"

$AppName = Read-Host "Application name"
$RedirectUris = Read-Host "Redirect URI(s), comma-separated"

$ClientId = [guid]::NewGuid().ToString()
$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
$ClientSecret = [BitConverter]::ToString($bytes).Replace("-", "").ToLower()

$sha256 = [System.Security.Cryptography.SHA256]::Create()
$hashBytes = $sha256.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($ClientSecret))
$ClientSecretHash = [BitConverter]::ToString($hashBytes).Replace("-", "").ToLower()

$uriArray = $RedirectUris.Split(",") | ForEach-Object { $_.Trim() }
$uriJson = ($uriArray | ForEach-Object { "`"$_`"" }) -join ","

$configJson = "{`"client_id`":`"$ClientId`",`"client_secret_hash`":`"$ClientSecretHash`",`"redirect_uris`":[$uriJson],`"allowed_scopes`":[`"openid`",`"profile`",`"email`"]}"

$sql = "INSERT INTO applications (id, name, protocol, config) VALUES (gen_random_uuid(), '$AppName', 'oidc', '$configJson');"

Write-Host ""
Write-Host "client_id:     $ClientId"
Write-Host "client_secret: $ClientSecret   (save this now, it is not stored in plaintext)"
Write-Host ""

if ($env:IDPFORGE_DB_DRIVER -and $env:IDPFORGE_DB_DRIVER -ne "postgres") {
    Write-Host "IDPFORGE_DB_DRIVER=$($env:IDPFORGE_DB_DRIVER) is not auto-applied by this script."
    Write-Host "Run the following against your database (adjust UUID generation for your dialect):"
    Write-Host $sql
    exit 0
}

$psql = Get-Command psql -ErrorAction SilentlyContinue
if ($psql -and $env:IDPFORGE_DB_DSN) {
    & psql $env:IDPFORGE_DB_DSN -c $sql
    Write-Host "Application registered."
} else {
    Write-Host "psql not found or IDPFORGE_DB_DSN not set. Run this SQL manually:"
    Write-Host $sql
}
