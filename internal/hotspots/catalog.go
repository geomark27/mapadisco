// Package hotspots define el catálogo de "sospechosos habituales" de consumo de disco
// en Windows y los evalúa contra el resultado de un escaneo.
//
// Para agregar un sospechoso nuevo basta con añadir una entrada a Catalog.
package hotspots

// Hotspot describe una ubicación conocida que suele acumular espacio.
type Hotspot struct {
	Label    string
	Patterns []string // rutas relativas a la raíz de la unidad; admiten comodines (* ?)
	Hint     string   // qué hacer al respecto (vacío si no hay recomendación)
}

// Match es un sospechoso encontrado en el disco.
type Match struct {
	Hotspot
	Size   int64
	Paths  []string // rutas concretas que coincidieron, con tamaño > 0
	Denied bool     // existe pero no se pudo leer
}

// Catalog es la lista de sospechosos que se revisan al escanear una unidad completa.
var Catalog = []Hotspot{
	// Sistema
	{"Hibernación (hiberfil.sys)", []string{`hiberfil.sys`}, "Si no usas hibernación: powercfg /h off (admin)"},
	{"Archivo de paginación", []string{`pagefile.sys`, `swapfile.sys`}, "Lo gestiona Windows; no borrar"},
	{"Puntos de restauración", []string{`System Volume Information`}, "Ver con: vssadmin list shadowstorage (admin)"},
	{"Windows.old (instalación anterior)", []string{`Windows.old`}, "Liberador de espacio > Instalaciones anteriores de Windows"},
	{"Caché de Windows Update", []string{`Windows\SoftwareDistribution\Download`}, "Liberador de espacio > Limpieza de Windows Update"},
	{"Almacén WinSxS", []string{`Windows\WinSxS`}, "Sobreestimado por enlaces duros. Dism /Online /Cleanup-Image /StartComponentCleanup"},
	{"Windows Installer (caché)", []string{`Windows\Installer`}, "No borrar a mano"},
	{"Optimización de distribución", []string{`Windows\ServiceProfiles\NetworkService\AppData\Local\Microsoft\Windows\DeliveryOptimization`}, "Liberador de espacio > Archivos de optimización de distribución"},
	{"Temporales del sistema", []string{`Windows\Temp`}, "Se libera con: mapadisco -clean -admin"},
	{"Volcados de memoria", []string{`Windows\MEMORY.DMP`, `Windows\Minidump`, `Users\*\AppData\Local\CrashDumps`, `ProgramData\Microsoft\Windows\WER`}, "Se libera con: mapadisco -clean"},
	{"Papelera de reciclaje", []string{`$Recycle.Bin`}, "Se vacía con: mapadisco -clean"},

	// Virtualización
	{"Discos virtuales de WSL", []string{`Users\*\AppData\Local\Packages\*\LocalState\*.vhdx`, `Users\*\AppData\Local\wsl\*\*.vhdx`}, "No se reducen solos: wsl --shutdown y compactar con Optimize-VHD o diskpart"},
	{"Docker Desktop", []string{`Users\*\AppData\Local\Docker`, `ProgramData\DockerDesktop`}, "docker system prune; luego compactar el .vhdx"},
	{"Hyper-V / VMs", []string{`ProgramData\Microsoft\Windows\Virtual Hard Disks`, `Users\Public\Documents\Hyper-V`, `Users\*\VirtualBox VMs`}, ""},

	// Juegos y GPU
	{"Caché de shaders (GPU / DirectX)", []string{`Users\*\AppData\Local\NVIDIA\DXCache`, `Users\*\AppData\Local\NVIDIA\GLCache`, `Users\*\AppData\Local\D3DSCache`, `Users\*\AppData\Local\AMD\DxCache`, `Users\*\AppData\Local\AMD\DxcCache`}, "Se libera con: mapadisco -clean (se regenera al jugar). Para que no vuelva a crecer: Panel de NVIDIA > Tamaño de caché del sombreador"},
	{"Juegos Xbox / Microsoft Store", []string{`XboxGames`, `Program Files\WindowsApps`}, "Desinstalar desde la app Xbox / Configuración > Aplicaciones"},
	{"Steam", []string{`Program Files (x86)\Steam\steamapps`}, "Revisa steamapps\\common y shadercache (restos de juegos)"},
	{"Epic Games", []string{`Program Files\Epic Games`, `ProgramData\Epic`}, ""},

	// Desarrollo
	{"Android SDK / emuladores", []string{`Users\*\AppData\Local\Android\Sdk`, `Users\*\.android\avd`}, "Borra AVDs e imágenes de sistema que no uses"},
	{"Caché npm / pnpm / yarn", []string{`Users\*\AppData\Local\npm-cache`, `Users\*\AppData\Local\pnpm`, `Users\*\AppData\Local\pnpm-cache`, `Users\*\AppData\Local\Yarn\Cache`}, "npm y yarn: mapadisco -clean · pnpm: pnpm store prune"},
	{"Caché pip / conda", []string{`Users\*\AppData\Local\pip\cache`, `Users\*\anaconda3\pkgs`, `Users\*\miniconda3\pkgs`}, "pip: mapadisco -clean · conda: conda clean -a"},
	{"Paquetes NuGet / Gradle / Maven", []string{`Users\*\.nuget\packages`, `Users\*\.gradle`, `Users\*\.m2`}, ""},
	{"Rust / Cargo", []string{`Users\*\.cargo`, `Users\*\.rustup`}, ""},
	{"Composer", []string{`Users\*\AppData\Local\Composer`}, "composer clear-cache"},

	// Usuario
	{"Temporales de usuario", []string{`Users\*\AppData\Local\Temp`}, "Se libera con: mapadisco -clean"},
	{"Navegadores (perfil + caché)", []string{`Users\*\AppData\Local\Google\Chrome\User Data`, `Users\*\AppData\Local\Microsoft\Edge\User Data`, `Users\*\AppData\Local\BraveSoftware`}, "Borrar datos de navegación > caché"},
	{"OneDrive (copia local)", []string{`Users\*\OneDrive*`}, "Clic derecho > Liberar espacio"},
	{"Descargas", []string{`Users\*\Downloads`}, "Revisa instaladores y .zip viejos"},
}
