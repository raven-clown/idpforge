param(
    [Parameter(Mandatory=$true)]
    [string]$BackupFile
)

$ErrorActionPreference = "Stop"
$Driver = if ($env:IDPFORGE_DB_DRIVER) { $env:IDPFORGE_DB_DRIVER } else { "postgres" }

switch ($Driver) {
    "postgres" {
        & pg_restore -d $env:IDPFORGE_DB_DSN --clean --if-exists $BackupFile
    }
    "mysql" {
        Get-Content $BackupFile | & mysql
    }
    "sqlite" {
        $Src = $env:IDPFORGE_DB_DSN -replace "^file:", ""
        Copy-Item $BackupFile $Src -Force
    }
    default {
        Write-Error "Restore for $Driver is not automated; restore $BackupFile manually."
        exit 1
    }
}

Write-Host "Restore complete."
