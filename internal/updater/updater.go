// Package updater busca, descarga e instala las versiones publicadas de mapadisco en su
// repositorio de GitHub. Un ejecutable solo se instala si su SHA-256 coincide con el
// checksums.txt del mismo release. Es portable: las pruebas usan un GitHub simulado.
package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Repo es el repositorio donde se publican las versiones.
const Repo = "geomark27/mapadisco"

// Asset es el ejecutable que publica cada release (scripts/release.sh).
const Asset = "mapadisco-windows-amd64.exe"

// ErrNoReleases indica que el repositorio todavía no tiene versiones publicadas.
var ErrNoReleases = errors.New("todavía no hay versiones publicadas")

// maxBinary limita una descarga, para que una URL equivocada no llene el disco.
const maxBinary = 100 << 20

// Client habla con GitHub; las pruebas apuntan API y Download a un servidor simulado.
type Client struct {
	API      string // https://api.github.com
	Download string // https://github.com
	HTTP     *http.Client
	Agent    string // User-Agent (GitHub lo exige en su API)
}

// New devuelve un cliente para github.com.
func New(version string) *Client {
	return &Client{
		API:      "https://api.github.com",
		Download: "https://github.com",
		HTTP:     http.DefaultClient,
		Agent:    "mapadisco/" + version,
	}
}

// Latest devuelve el tag de la última versión publicada (sin pre-releases).
func (c *Client) Latest(ctx context.Context) (string, error) {
	body, err := c.get(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", c.API, Repo), 1<<20)
	var status statusError
	if errors.As(err, &status) && status == http.StatusNotFound {
		return "", ErrNoReleases
	}
	if err != nil {
		return "", err
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.TagName == "" {
		return "", errors.New("respuesta inesperada de GitHub al buscar la última versión")
	}
	return r.TagName, nil
}

// Fetch descarga el ejecutable de version y lo devuelve solo si su SHA-256 coincide
// con el checksums.txt del release.
func (c *Client) Fetch(ctx context.Context, version string) ([]byte, error) {
	base := fmt.Sprintf("%s/%s/releases/download/%s/", c.Download, Repo, version)
	sums, err := c.get(ctx, base+"checksums.txt", 1<<20)
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener checksums.txt: %w", err)
	}
	want, err := checksum(sums, Asset)
	if err != nil {
		return nil, err
	}
	data, err := c.get(ctx, base+Asset, maxBinary)
	if err != nil {
		return nil, fmt.Errorf("no se pudo descargar %s: %w", Asset, err)
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return nil, fmt.Errorf("el SHA-256 de %s no coincide con checksums.txt: no se instala", Asset)
	}
	return data, nil
}

// ReleaseURL es la página de una versión en GitHub.
func ReleaseURL(version string) string {
	return fmt.Sprintf("https://github.com/%s/releases/tag/%s", Repo, version)
}

// checksum lee el hash de asset en un archivo con el formato de sha256sum.
func checksum(sums []byte, asset string) (string, error) {
	sc := bufio.NewScanner(strings.NewReader(string(sums)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == asset {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("checksums.txt no tiene el hash de %s", asset)
}

func (c *Client) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
	if c.Agent != "" {
		req.Header.Set("User-Agent", c.Agent)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("la descarga es más grande de lo esperado")
	}
	return data, nil
}

type statusError int

func (s statusError) Error() string { return fmt.Sprintf("GitHub respondió %d", int(s)) }

// IsRelease indica si v es una versión publicada (vMAJOR.MINOR.PATCH) y no una compilación
// de desarrollo como "dev" o "f6099e4-dirty".
func IsRelease(v string) bool {
	_, ok := semver(v)
	return ok
}

// IsNewer indica si candidate es una versión vMAJOR.MINOR.PATCH posterior a base. Si alguna
// no tiene ese formato (una compilación de desarrollo), nunca es más nueva.
func IsNewer(candidate, base string) bool {
	c, ok1 := semver(candidate)
	b, ok2 := semver(base)
	if !ok1 || !ok2 {
		return false
	}
	for i := range c {
		if c[i] != b[i] {
			return c[i] > b[i]
		}
	}
	return false
}

func semver(v string) ([3]int, bool) {
	var out [3]int
	if !strings.HasPrefix(v, "v") {
		return out, false
	}
	parts := strings.Split(v[1:], ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Replace escribe data sobre el ejecutable exe. La versión nueva se escribe al lado y se
// renombra, así un fallo nunca deja medio ejecutable. Windows no deja sobrescribir un .exe en
// uso pero sí renombrarlo: el actual se aparta como .old y installer.CleanOld lo borra al
// arrancar la próxima vez.
func Replace(exe string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".mapadisco-update-*")
	if err != nil {
		return fmt.Errorf("no se pudo escribir junto a %s: %w", exe, err)
	}
	name := tmp.Name()
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(name, 0o755)
	}
	if err != nil {
		os.Remove(name)
		return fmt.Errorf("no se pudo escribir la nueva versión: %w", err)
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(name)
		return fmt.Errorf("no se pudo apartar el ejecutable actual: %w", err)
	}
	if err := os.Rename(name, exe); err != nil {
		os.Rename(old, exe) // deja todo como estaba
		os.Remove(name)
		return fmt.Errorf("no se pudo reemplazar el ejecutable: %w", err)
	}
	return nil
}
