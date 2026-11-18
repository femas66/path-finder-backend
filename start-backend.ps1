# Menyalakan backend Path Finder (PostgreSQL + Go) dalam satu langkah.
#
# Pemakaian (dari folder path-finder-backend):
#   .\start-backend.ps1
#
# Skrip ini:
#   1. Menyalakan Docker Desktop bila daemon-nya belum jalan
#   2. Menyalakan container PostgreSQL (docker compose up -d postgres)
#   3. Menunggu sampai PostgreSQL benar-benar siap
#   4. Menjalankan server Go di port 3000 (foreground, Ctrl+C untuk berhenti)

$ErrorActionPreference = 'Stop'
$backendDir = $PSScriptRoot
Set-Location $backendDir

function Write-Step($message) {
    Write-Host ""
    Write-Host "==> $message" -ForegroundColor Cyan
}

function Write-Ok($message) {
    Write-Host "    OK  $message" -ForegroundColor Green
}

function Write-Fail($message) {
    Write-Host "    !!  $message" -ForegroundColor Red
}

# Docker menulis progres ke stderr. PowerShell memperlakukannya sebagai error
# fatal saat $ErrorActionPreference = 'Stop', jadi native command dijalankan
# lewat helper ini yang menonaktifkan perilaku tersebut sementara.
# Mengembalikan objek berisi ExitCode dan Output (gabungan stdout + stderr).
function Invoke-Native {
    param(
        [Parameter(Mandatory)][scriptblock]$Command
    )

    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $lines = @(& $Command 2>&1 | ForEach-Object { "$_" })
        return [pscustomobject]@{
            ExitCode = $LASTEXITCODE
            Output   = ($lines -join "`n").Trim()
        }
    }
    finally {
        $ErrorActionPreference = $previous
    }
}

# ---------------------------------------------------------------- 1. Docker
Write-Step "Memeriksa Docker"

$dockerReady = (Invoke-Native { docker ps }).ExitCode -eq 0

if (-not $dockerReady) {
    Write-Host "    Docker daemon belum jalan, mencoba menyalakan Docker Desktop..." -ForegroundColor Yellow

    $dockerExe = @(
        "$env:LOCALAPPDATA\Programs\DockerDesktop\Docker Desktop.exe",
        "$env:ProgramFiles\Docker\Docker\Docker Desktop.exe"
    ) | Where-Object { Test-Path $_ } | Select-Object -First 1

    if (-not $dockerExe) {
        Write-Fail "Docker Desktop tidak ditemukan. Pasang dulu dari https://www.docker.com/products/docker-desktop"
        exit 1
    }

    Start-Process $dockerExe

    $ready = $false
    for ($i = 1; $i -le 60; $i++) {
        Start-Sleep -Seconds 2
        if ((Invoke-Native { docker ps }).ExitCode -eq 0) { $ready = $true; break }
        Write-Host "    menunggu Docker Desktop... ($($i * 2)s)" -ForegroundColor DarkGray
    }

    if (-not $ready) {
        Write-Fail "Docker Desktop tidak siap setelah 2 menit. Buka manual lalu jalankan skrip ini lagi."
        exit 1
    }
}
Write-Ok "Docker siap"

# ------------------------------------------------------------- 2. PostgreSQL
Write-Step "Menyalakan PostgreSQL"

$compose = Invoke-Native { docker compose up -d postgres }
if ($compose.ExitCode -ne 0) {
    Write-Fail "Gagal menjalankan 'docker compose up -d postgres'."
    Write-Host $compose.Output -ForegroundColor DarkGray
    exit 1
}

$healthy = $false
for ($i = 1; $i -le 30; $i++) {
    $inspect = Invoke-Native { docker inspect --format '{{.State.Health.Status}}' path_finder_postgres }
    if ($inspect.ExitCode -eq 0 -and $inspect.Output -eq 'healthy') {
        $healthy = $true
        break
    }
    Start-Sleep -Seconds 2
    Write-Host "    menunggu PostgreSQL siap... ($($i * 2)s)" -ForegroundColor DarkGray
}

if (-not $healthy) {
    Write-Fail "PostgreSQL tidak menjadi healthy. Cek dengan: docker compose logs postgres"
    exit 1
}
Write-Ok "PostgreSQL healthy di port 5432"

# ------------------------------------------------------------- 3. Port check
Write-Step "Memeriksa port 3000"

$inUse = Get-NetTCPConnection -LocalPort 3000 -State Listen -ErrorAction SilentlyContinue
if ($inUse) {
    $owner = Get-Process -Id $inUse[0].OwningProcess -ErrorAction SilentlyContinue
    Write-Fail "Port 3000 sudah dipakai oleh '$($owner.ProcessName)' (PID $($inUse[0].OwningProcess))."
    Write-Host "    Hentikan dulu:  Stop-Process -Id $($inUse[0].OwningProcess) -Force" -ForegroundColor Yellow
    exit 1
}
Write-Ok "Port 3000 bebas"

# ------------------------------------------------------------ 4. Go backend
Write-Step "Menjalankan server Go di http://localhost:3000"
Write-Host "    Biarkan terminal ini terbuka. Tekan Ctrl+C untuk menghentikan backend." -ForegroundColor Yellow
Write-Host ""

go run ./cmd/web
