param()

$ErrorActionPreference = "Stop"

$BackupDir = if ($env:IDPFORGE_BACKUP_DIR) { $env:IDPFORGE_BACKUP_DIR } else { "$env:ProgramData\idpforge\data\backups" }
$RetentionDays = if ($env:IDPFORGE_BACKUP_RETENTION_DAYS) { [int]$env:IDPFORGE_BACKUP_RETENTION_DAYS } else { 14 }
$Timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null

$Driver = if ($env:IDPFORGE_DB_DRIVER) { $env:IDPFORGE_DB_DRIVER } else { "postgres" }

switch ($Driver) {
    "postgres" {
        $Out = Join-Path $BackupDir "idpforge-$Timestamp.dump"
        & pg_dump $env:IDPFORGE_DB_DSN -Fc -f $Out
    }
    "mysql" {
        $Out = Join-Path $BackupDir "idpforge-$Timestamp.sql"
        & mysqldump --result-file=$Out
    }
    "mssql" {
        Write-Host "MSSQL: run BACKUP DATABASE via sqlcmd; not automated here."
        exit 1
    }
    "sqlite" {
        $Src = $env:IDPFORGE_DB_DSN -replace "^file:", ""
        $Out = Join-Path $BackupDir "idpforge-$Timestamp.sqlite"
        & sqlite3 $Src ".backup '$Out'"
    }
    default {
        Write-Error "Unknown IDPFORGE_DB_DRIVER"
        exit 1
    }
}

Write-Host "Backup written to $Out"

Get-ChildItem $BackupDir -File |
    Where-Object { $_.LastWriteTimeUtc -lt (Get-Date).ToUniversalTime().AddDays(-$RetentionDays) } |
    ForEach-Object {
        Write-Host "Removing old backup $($_.FullName)"
        Remove-Item $_.FullName -Force
    }
