param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrEmpty($Version)) {
    $gitVersion = git describe --tags --always --dirty 2>$null
    if ($LASTEXITCODE -eq 0 -and $gitVersion) {
        $Version = $gitVersion
    } else {
        $Version = "dev"
    }
}

$Binary = "idpforge-server"
$Dist = "dist"
$Ldflags = "-s -w -X main.version=$Version"

$Platforms = @(
    @{ OS = "linux";   Arch = "amd64" },
    @{ OS = "linux";   Arch = "arm64" },
    @{ OS = "windows"; Arch = "amd64" },
    @{ OS = "windows"; Arch = "arm64" }
)

New-Item -ItemType Directory -Force -Path $Dist | Out-Null

foreach ($p in $Platforms) {
    $ext = if ($p.OS -eq "windows") { ".exe" } else { "" }
    $out = "$Dist/$Binary-$($p.OS)-$($p.Arch)$ext"

    Write-Host "Building $out"
    $env:CGO_ENABLED = "0"
    $env:GOOS = $p.OS
    $env:GOARCH = $p.Arch
    go build -ldflags $Ldflags -o $out ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "build failed for $($p.OS)/$($p.Arch)" }

    $archiveName = "$Dist/$Binary-$($p.OS)-$($p.Arch).tar.gz"
    tar -czf $archiveName -C $Dist "$Binary-$($p.OS)-$($p.Arch)$ext"
}

Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "Done. Artifacts in $Dist/"
