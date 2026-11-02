package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGitHub simula la API y las descargas de un release con el ejecutable exe.
func fakeGitHub(t *testing.T, latest string, exe []byte, sums string) (*Client, *[]string) {
	t.Helper()
	var agents []string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		agents = append(agents, r.Header.Get("User-Agent"))
		if latest == "" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"tag_name":%q,"name":"x"}`, latest)
	})
	base := "/" + Repo + "/releases/download/" + latest + "/"
	mux.HandleFunc(base+"checksums.txt", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, sums) })
	mux.HandleFunc(base+Asset, func(w http.ResponseWriter, r *http.Request) { w.Write(exe) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := New("v0.1.0")
	c.API, c.Download, c.HTTP = srv.URL, srv.URL, srv.Client()
	return c, &agents
}

func sumLine(data []byte, name string) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]) + "  " + name + "\n"
}

func TestLatest(t *testing.T) {
	c, agents := fakeGitHub(t, "v0.4.0", nil, "")
	got, err := c.Latest(context.Background())
	if err != nil || got != "v0.4.0" {
		t.Fatalf("Latest = %q, %v", got, err)
	}
	if (*agents)[0] != "mapadisco/v0.1.0" {
		t.Errorf("User-Agent = %q", (*agents)[0])
	}
}

func TestLatestSinReleases(t *testing.T) {
	c, _ := fakeGitHub(t, "", nil, "")
	if _, err := c.Latest(context.Background()); !errors.Is(err, ErrNoReleases) {
		t.Fatalf("err = %v, quiero ErrNoReleases", err)
	}
}

func TestFetchVerificaElSHA256(t *testing.T) {
	exe := []byte("ejecutable nuevo")
	sums := sumLine([]byte("otro"), "otro.exe") + sumLine(exe, Asset)
	c, _ := fakeGitHub(t, "v0.4.0", exe, sums)
	got, err := c.Fetch(context.Background(), "v0.4.0")
	if err != nil || string(got) != string(exe) {
		t.Fatalf("Fetch = %q, %v", got, err)
	}
}

func TestFetchRechazaUnSHA256Distinto(t *testing.T) {
	c, _ := fakeGitHub(t, "v0.4.0", []byte("alterado"), sumLine([]byte("original"), Asset))
	_, err := c.Fetch(context.Background(), "v0.4.0")
	if err == nil || !strings.Contains(err.Error(), "no coincide") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchSinHashEnChecksums(t *testing.T) {
	c, _ := fakeGitHub(t, "v0.4.0", []byte("x"), sumLine([]byte("x"), "otro.exe"))
	if _, err := c.Fetch(context.Background(), "v0.4.0"); err == nil {
		t.Fatal("sin hash del ejecutable no debe instalar")
	}
}

func TestChecksumFormatoBinario(t *testing.T) {
	got, err := checksum([]byte("ABCDEF *"+Asset+"\n"), Asset)
	if err != nil || got != "abcdef" {
		t.Fatalf("checksum = %q, %v", got, err)
	}
}

func TestIsNewerEIsRelease(t *testing.T) {
	cases := []struct {
		cand, base string
		want       bool
	}{
		{"v0.4.0", "v0.3.0", true},
		{"v0.10.0", "v0.9.9", true},
		{"v1.0.0", "v0.99.99", true},
		{"v0.3.0", "v0.3.0", false},
		{"v0.2.9", "v0.3.0", false},
		{"v0.4.0", "dev", false},
		{"v0.4.0", "f6099e4-dirty", false},
		{"v0.4.0", "v0.3.0-2-gabc1234", false},
		{"0.4.0", "v0.3.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.cand, c.base); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, quiero %v", c.cand, c.base, got, c.want)
		}
	}
	if !IsRelease("v0.3.0") || IsRelease("dev") || IsRelease("abc1234-dirty") {
		t.Error("IsRelease")
	}
}

func TestReplace(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "mapadisco.exe")
	os.WriteFile(exe, []byte("v0.3.0"), 0o755)
	if err := Replace(exe, []byte("v0.4.0")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "v0.4.0" {
		t.Errorf("ejecutable = %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "v0.3.0" {
		t.Errorf(".old = %q", b)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), ".mapadisco-update-*")); len(left) > 0 {
		t.Errorf("quedaron temporales: %v", left)
	}
}

func TestReplaceSinEjecutableNoDejaRastros(t *testing.T) {
	dir := t.TempDir()
	if err := Replace(filepath.Join(dir, "no-existe.exe"), []byte("x")); err == nil {
		t.Fatal("debe fallar si no hay ejecutable que reemplazar")
	}
	if left, _ := os.ReadDir(dir); len(left) > 0 {
		t.Errorf("quedaron archivos: %v", left)
	}
}
