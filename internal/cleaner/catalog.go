// Package cleaner libera espacio borrando solo categorías conocidas que Windows o los programas
// regeneran solos (temporales, cachés, papelera, volcados). Lo que no se puede borrar con
// seguridad desde aquí se describe como guía manual: a dónde ir y qué hacer.
//
// Las rutas usan variables de entorno ($LOCALAPPDATA, $TEMP, ${ProgramFiles(x86)}…) y pueden
// tener comodines. El catálogo es datos: para agregar una categoría basta una entrada.
package cleaner

import (
	"fmt"
	"slices"
	"strings"
)

// Kind indica cómo se limpia una categoría.
type Kind int

const (
	// Contents borra el contenido de cada carpeta (nunca la carpeta en sí) o el archivo indicado.
	Contents Kind = iota
	// RecycleBin vacía la Papelera con la API de Windows.
	RecycleBin
)

// Target es una categoría que mapadisco puede limpiar.
type Target struct {
	ID     string
	Label  string
	Note   string   // qué implica limpiarla
	Paths  []string // carpetas cuyo contenido se borra, o archivos sueltos
	Kind   Kind
	Admin  bool   // parte de su contenido suele requerir Administrador
	Manual string // a dónde ir si Windows no deja borrarla
}

// Guide es lo que mapadisco no borra por seguridad: se mide y se explica cómo liberarlo a mano.
type Guide struct {
	Label string
	Paths []string // para medir cuánto ocupa; puede estar vacío
	Steps []string
}

// Auto son las categorías que -clean puede borrar.
var Auto = []Target{
	{
		ID: "shaders", Label: "Caché de shaders (GPU / DirectX)",
		Note: "se regenera sola; los primeros minutos de cada juego pueden ir algo más lentos",
		Paths: []string{
			`$LOCALAPPDATA\NVIDIA\DXCache`, `$LOCALAPPDATA\NVIDIA\GLCache`, `$LOCALAPPDATA\D3DSCache`,
			`$LOCALAPPDATA\AMD\DxCache`, `$LOCALAPPDATA\AMD\DxcCache`, `$LOCALAPPDATA\Intel\ShaderCache`,
		},
		Manual: "cierra los juegos y el panel de la tarjeta gráfica y vuelve a intentar; o Liberador de espacio " +
			"(cleanmgr) > «Caché de sombreador DirectX»",
	},
	{
		ID: "temp", Label: "Temporales de tu usuario",
		Note:   "archivos temporales que dejan los programas",
		Paths:  []string{`$TEMP`},
		Manual: "Configuración > Sistema > Almacenamiento > Archivos temporales (start ms-settings:storagesense)",
	},
	{
		ID: "temp-sistema", Label: "Temporales de Windows",
		Note:   "archivos temporales del sistema",
		Paths:  []string{`$SystemRoot\Temp`},
		Admin:  true,
		Manual: "mapadisco -clean -admin; o Configuración > Sistema > Almacenamiento > Archivos temporales (start ms-settings:storagesense)",
	},
	{
		ID: "papelera", Label: "Papelera de reciclaje",
		Note:   "lo que ya borraste; después no se puede recuperar",
		Kind:   RecycleBin,
		Manual: "clic derecho en la Papelera del escritorio > Vaciar Papelera de reciclaje",
	},
	{
		ID: "volcados", Label: "Volcados de memoria e informes de errores",
		Note: "diagnósticos de programas o de Windows que fallaron",
		Paths: []string{
			`$LOCALAPPDATA\CrashDumps`, `$LOCALAPPDATA\Microsoft\Windows\WER\ReportArchive`,
			`$LOCALAPPDATA\Microsoft\Windows\WER\ReportQueue`, `$ProgramData\Microsoft\Windows\WER\ReportArchive`,
			`$ProgramData\Microsoft\Windows\WER\ReportQueue`, `$SystemRoot\Minidump`, `$SystemRoot\MEMORY.DMP`,
		},
		Admin: true,
		Manual: "mapadisco -clean -admin; o Liberador de espacio (cleanmgr) > «Limpiar archivos del sistema» > " +
			"«Archivos de volcado de memoria» e «Informes de errores»",
	},
	{
		ID: "dev", Label: "Cachés de desarrollo (npm, pip, yarn)",
		Note:   "paquetes descargados; se vuelven a descargar cuando hagan falta",
		Paths:  []string{`$LOCALAPPDATA\npm-cache`, `$LOCALAPPDATA\pip\cache`, `$LOCALAPPDATA\Yarn\Cache`},
		Manual: "npm cache clean --force · pip cache purge · yarn cache clean",
	},
}

// Manual son las guías para lo que mapadisco no borra por su cuenta.
var Manual = []Guide{
	{
		Label: "Windows Update e instalación anterior (Windows.old)",
		Paths: []string{`$SystemRoot\SoftwareDistribution\Download`, `$SystemDrive\Windows.old`},
		Steps: []string{
			"Configuración > Sistema > Almacenamiento > Archivos temporales: marca «Limpieza de Windows Update» " +
				"e «Instalaciones anteriores de Windows» (start ms-settings:storagesense)",
			"o: cleanmgr > «Limpiar archivos del sistema»",
		},
	},
	{
		Label: "Almacén de componentes (WinSxS)",
		Paths: []string{`$SystemRoot\WinSxS`},
		Steps: []string{
			"en una terminal como Administrador: Dism /Online /Cleanup-Image /StartComponentCleanup",
			"nunca borrar a mano; el tamaño que se ve está inflado por enlaces duros",
		},
	},
	{
		Label: "Puntos de restauración",
		Paths: []string{`$SystemDrive\System Volume Information`},
		Steps: []string{
			"SystemPropertiesProtection > Configurar: baja el «Uso máximo» o «Eliminar» los puntos anteriores",
			"para ver cuánto ocupan, como Administrador: vssadmin list shadowstorage",
		},
	},
	{
		Label: "Hibernación (hiberfil.sys)",
		Paths: []string{`$SystemDrive\hiberfil.sys`},
		Steps: []string{
			"si no usas hibernación ni inicio rápido, como Administrador: powercfg /h off (se revierte con powercfg /h on)",
		},
	},
	{
		Label: "Disco virtual de WSL (ext4.vhdx)",
		Paths: []string{`$LOCALAPPDATA\wsl\*\*.vhdx`, `$LOCALAPPDATA\Packages\*\LocalState\*.vhdx`},
		Steps: []string{
			"borra dentro de Linux lo que no necesites; el .vhdx no se encoge solo",
			"wsl --shutdown y, en PowerShell como Administrador: Optimize-VHD -Path \"<ruta del .vhdx>\" -Mode Full",
			"sin Hyper-V, con diskpart: select vdisk file=\"<ruta>\" · attach vdisk readonly · compact vdisk · detach vdisk",
		},
	},
	{
		Label: "Docker Desktop",
		Paths: []string{`$LOCALAPPDATA\Docker\wsl\disk\*.vhdx`, `$LOCALAPPDATA\Docker\wsl\data\*.vhdx`},
		Steps: []string{
			"docker system prune -a (borra imágenes y contenedores sin uso; revisa antes cuáles necesitas)",
			"luego compacta el .vhdx igual que el de WSL",
		},
	},
	{
		Label: "Juegos instalados",
		Paths: []string{`${ProgramFiles(x86)}\Steam\steamapps\common`, `$ProgramFiles\Epic Games`, `$SystemDrive\XboxGames`},
		Steps: []string{
			"desinstala desde su lanzador (Steam, Epic, app Xbox) o en Configuración > Aplicaciones",
			"revisa que no queden carpetas en steamapps\\common de juegos ya desinstalados",
		},
	},
	{
		Label: "Descargas",
		Paths: []string{`$USERPROFILE\Downloads`},
		Steps: []string{"revisa instaladores, .zip e .iso que ya no uses: explorer shell:Downloads"},
	},
	{
		Label: "Navegadores (caché)",
		Paths: []string{
			`$LOCALAPPDATA\Google\Chrome\User Data`, `$LOCALAPPDATA\Microsoft\Edge\User Data`,
			`$LOCALAPPDATA\BraveSoftware\Brave-Browser\User Data`,
		},
		Steps: []string{"en cada navegador: Ctrl+Shift+Supr > «Imágenes y archivos en caché» (no borres la carpeta a mano: es tu perfil)"},
	},
}

// IDs devuelve los identificadores de las categorías automáticas, en orden.
func IDs() []string {
	ids := make([]string, len(Auto))
	for i, t := range Auto {
		ids[i] = t.ID
	}
	return ids
}

// Select devuelve las categorías pedidas por ID ("shaders,temp"), en el orden del catálogo;
// con only vacío, todas.
func Select(only string) ([]Target, error) {
	if strings.TrimSpace(only) == "" {
		return Auto, nil
	}
	want := map[string]bool{}
	for _, id := range strings.Split(only, ",") {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" {
			continue
		}
		if !slices.Contains(IDs(), id) {
			return nil, fmt.Errorf("categoría desconocida en -only: %q (válidas: %s)", id, strings.Join(IDs(), ", "))
		}
		want[id] = true
	}
	var out []Target
	for _, t := range Auto {
		if want[t.ID] {
			out = append(out, t)
		}
	}
	return out, nil
}
