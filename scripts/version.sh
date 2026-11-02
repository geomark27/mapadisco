# Funciones de versión compartidas por release-prep.sh y release.sh. Se carga con source.

fail() { echo "✗ $*" >&2; exit 1; }
ok() { echo "✓ $*"; }

# last_version: el tag vX.Y.Z más alto, o v0.0.0 si no hay. No usa git describe: los tags están en los
# merge commits de main, que develop y sus ramas no contienen, y describe solo ve tags ancestros de HEAD.
last_version() {
	local v
	v=$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1)
	echo "${v:-v0.0.0}"
}

# next_version <última> <bump> <ver>: ver si se dio, o la última más el bump. Valida formato y orden.
next_version() {
	local last=$1 bump=$2 ver=$3 major minor patch
	if [ -z "$ver" ]; then
		IFS=. read -r major minor patch <<<"${last#v}"
		case "$bump" in
		patch) ver="v$major.$minor.$((patch + 1))" ;;
		minor) ver="v$major.$((minor + 1)).0" ;;
		major) ver="v$((major + 1)).0.0" ;;
		*) fail "indica bump=patch|minor|major o ver=vX.Y.Z" ;;
		esac
	fi
	[[ "$ver" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "versión inválida: $ver (vMAJOR.MINOR.PATCH)"
	[ "$ver" != "$last" ] && [ "$(printf '%s\n%s\n' "$last" "$ver" | sort -V | tail -1)" = "$ver" ] ||
		fail "$ver no es mayor que la última versión ($last)"
	echo "$ver"
}

# bump_type <última> <nueva>: patch, minor o major según qué parte cambió.
bump_type() {
	local a b
	IFS=. read -r -a a <<<"${1#v}"
	IFS=. read -r -a b <<<"${2#v}"
	if [ "${a[0]}" != "${b[0]}" ]; then echo major
	elif [ "${a[1]}" != "${b[1]}" ]; then echo minor
	else echo patch; fi
}

# devlog_dir: carpeta de la bitácora según .devlog.conf.
devlog_dir() {
	sed -nE 's/^[[:space:]]*DEVLOG_DIR[[:space:]]*=[[:space:]]*"?([^"#[:space:]]+)"?.*/\1/p' .devlog.conf 2>/dev/null | tail -n1
}
