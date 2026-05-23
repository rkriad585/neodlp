# build.ps1 - Cross-platform Go build script (Windows PowerShell)
# Builds neodlp binaries for all supported OS/arch combinations.

$ErrorActionPreference = 'Stop'

# ── Configuration ────────────────────────────────────────────────────────────
$BinaryName    = "neodlp"
$PublisherName = "rkriad585"
$PublisherEmail = "rkriad585@gmail.com"
$LdflagsPrefix = "neodlp/internal/version"

# ── Resolve version ─────────────────────────────────────────────────────────
$VersionFile = Join-Path $PSScriptRoot ".version"
if (Test-Path $VersionFile) {
    $Version = (Get-Content $VersionFile -Raw).Trim()
} else {
    $Version = "0.0.0"
    Write-Warning ".version file not found, defaulting to $Version"
}

# ── Resolve Git commit ──────────────────────────────────────────────────────
try {
    $Commit = (git rev-parse --short HEAD 2>$null).Trim()
} catch {
    $Commit = "unknown"
}
if ([string]::IsNullOrWhiteSpace($Commit)) { $Commit = "unknown" }

# ── Detect host architecture ────────────────────────────────────────────────
# Win32_Processor Architecture:
#   0 = x86, 1 = MIPS, 2 = Alpha, 3 = PowerPC,
#   5 = ARM, 6 = IA64, 9 = x64 (AMD64), 12 = ARM64
$procArch = (Get-CimInstance Win32_Processor | Select-Object -First 1).Architecture
switch ($procArch) {
    0  { $HostArch = "386"   }
    9  { $HostArch = "amd64" }
    12 { $HostArch = "arm64" }
    5  { $HostArch = "arm"   }
    default { $HostArch = "amd64" }
}

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║          neodlp Cross-Platform Builder           ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Version  : $Version" -ForegroundColor Yellow
Write-Host "  Commit   : $Commit" -ForegroundColor Yellow
Write-Host "  Publisher: $PublisherName <$PublisherEmail>" -ForegroundColor Yellow
Write-Host "  Host Arch: $HostArch (Win32_Processor=$procArch)" -ForegroundColor Yellow
Write-Host ""

# ── Target matrix ────────────────────────────────────────────────────────────
$Targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; Ext = ".exe" },
    @{ GOOS = "linux";   GOARCH = "amd64"; Ext = ""     },
    @{ GOOS = "linux";   GOARCH = "arm64"; Ext = ""     },
    @{ GOOS = "darwin";  GOARCH = "amd64"; Ext = ""     },
    @{ GOOS = "darwin";  GOARCH = "arm64"; Ext = ""     }
)

# ── Prepare output directory ─────────────────────────────────────────────────
$OutDir = Join-Path $PSScriptRoot "bin"
if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

# ── Build ldflags ────────────────────────────────────────────────────────────
$Ldflags = @(
    "-s", "-w",
    "-X", "$LdflagsPrefix.Version=$Version",
    "-X", "$LdflagsPrefix.Commit=$Commit",
    "-X", "$LdflagsPrefix.PublisherName=$PublisherName",
    "-X", "$LdflagsPrefix.PublisherEmail=$PublisherEmail"
) -join " "

# ── Build loop ───────────────────────────────────────────────────────────────
$Built   = 0
$Failed  = 0
$Total   = $Targets.Count
$StartTime = Get-Date

foreach ($t in $Targets) {
    $env:GOOS   = $t.GOOS
    $env:GOARCH = $t.GOARCH
    $env:CGO_ENABLED = "0"

    $OutName = "{0}-{1}-{2}{3}" -f $BinaryName, $t.GOOS, $t.GOARCH, $t.Ext
    $OutPath = Join-Path $OutDir $OutName

    Write-Host "  [$($Built + $Failed + 1)/$Total] Building $OutName ... " -NoNewline -ForegroundColor White

    try {
        go build -trimpath -ldflags "$Ldflags" -o $OutPath .
        $size = [math]::Round((Get-Item $OutPath).Length / 1MB, 2)
        Write-Host "OK (${size} MB)" -ForegroundColor Green
        $Built++
    } catch {
        Write-Host "FAILED" -ForegroundColor Red
        Write-Warning "  Error: $_"
        $Failed++
    }
}

# ── Cleanup environment ─────────────────────────────────────────────────────
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

# ── Summary ──────────────────────────────────────────────────────────────────
$Duration = (Get-Date) - $StartTime
Write-Host ""
Write-Host "══════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Build complete in $([math]::Round($Duration.TotalSeconds, 1))s" -ForegroundColor Cyan
Write-Host "  Success: $Built / $Total" -ForegroundColor $(if ($Failed -eq 0) { "Green" } else { "Yellow" })
if ($Failed -gt 0) {
    Write-Host "  Failed : $Failed / $Total" -ForegroundColor Red
}
Write-Host "  Output : $OutDir" -ForegroundColor Cyan
Write-Host "══════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

if ($Failed -gt 0) { exit 1 }
