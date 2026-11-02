# Instala mapadisco en Windows: descarga el ejecutable del último release, verifica su
# SHA-256 contra checksums.txt y deja que el propio mapadisco se instale (mapadisco -install):
# lo copia a %LOCALAPPDATA%\Programs\mapadisco y agrega esa carpeta al PATH del usuario.
#   irm https://raw.githubusercontent.com/geomark27/mapadisco/main/scripts/install.ps1 | iex
# $env:MAPADISCO_VERSION = "vX.Y.Z" instala esa versión.

& {
	$ErrorActionPreference = "Stop"
	$repo = "geomark27/mapadisco"
	$asset = "mapadisco-windows-amd64.exe"

	if (-not [Environment]::Is64BitOperatingSystem) { throw "mapadisco se publica solo para Windows de 64 bits" }
	if ($env:MAPADISCO_VERSION) {
		$base = "https://github.com/$repo/releases/download/$($env:MAPADISCO_VERSION)"
	} else {
		$base = "https://github.com/$repo/releases/latest/download"
	}

	$tmp = Join-Path ([IO.Path]::GetTempPath()) ("mapadisco-" + [Guid]::NewGuid())
	New-Item -ItemType Directory -Force $tmp | Out-Null
	try {
		[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
		Write-Host "Descargando $asset..."
		Invoke-WebRequest -UseBasicParsing "$base/$asset" -OutFile (Join-Path $tmp $asset)
		Invoke-WebRequest -UseBasicParsing "$base/checksums.txt" -OutFile (Join-Path $tmp "checksums.txt")

		$want = $null
		foreach ($line in Get-Content (Join-Path $tmp "checksums.txt")) {
			$parts = $line -split '\s+'
			if ($parts.Count -ge 2 -and $parts[1].TrimStart('*') -eq $asset) { $want = $parts[0].ToLower() }
		}
		if (-not $want) { throw "checksums.txt no trae $asset" }
		$got = (Get-FileHash (Join-Path $tmp $asset) -Algorithm SHA256).Hash.ToLower()
		if ($got -ne $want) { throw "el SHA-256 no coincide con checksums.txt: no se instaló nada" }
		Write-Host "OK: SHA-256 verificado"

		& (Join-Path $tmp $asset) -install
		if ($LASTEXITCODE -ne 0) { throw "la instalación falló (código $LASTEXITCODE)" }
	} finally {
		Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
	}

	$dir = Join-Path $env:LOCALAPPDATA "Programs\mapadisco"
	if (($env:Path -split ';') -notcontains $dir) { $env:Path += ";$dir" }
	& (Join-Path $dir "mapadisco.exe") -version
}
