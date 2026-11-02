# Instalador de mapadisco (por usuario, no requiere Administrador)
param([switch]$Uninstall)
$ErrorActionPreference = 'Stop'
$dest = Join-Path $env:LOCALAPPDATA 'Programs\mapadisco'
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not $userPath) { $userPath = '' }
$parts = $userPath -split ';' | Where-Object { $_ -ne '' }

if ($Uninstall) {
    if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
    $new = ($parts | Where-Object { $_ -ine $dest }) -join ';'
    [Environment]::SetEnvironmentVariable('Path', $new, 'User')
    Write-Host 'mapadisco desinstalado.' -ForegroundColor Green
    exit 0
}

$src = Join-Path $PSScriptRoot 'mapadisco.exe'
if (-not (Test-Path $src)) { throw "No se encontro mapadisco.exe junto al instalador ($PSScriptRoot)" }
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Copy-Item -Force $src $dest
Unblock-File -Path (Join-Path $dest 'mapadisco.exe') -ErrorAction SilentlyContinue

if (-not ($parts | Where-Object { $_ -ieq $dest })) {
    [Environment]::SetEnvironmentVariable('Path', (($parts + $dest) -join ';'), 'User')
    Write-Host "Agregado al PATH de usuario: $dest"
}
Write-Host ''
Write-Host 'mapadisco instalado.' -ForegroundColor Green
Write-Host 'Abre una NUEVA terminal y ejecuta:  mapadisco        (o  mapadisco -admin  para ver todo)'
