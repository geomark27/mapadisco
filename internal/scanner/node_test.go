package scanner

import "testing"

// tree construye: C:\ → Users → Marcos → AppData ; C:\ → Windows
func tree() *Node {
	root := &Node{Name: `C:\`, Size: 10, Files: 1}
	users := &Node{Name: "Users", Parent: root}
	marcos := &Node{Name: "Marcos", Parent: users, Size: 5, Files: 2}
	appdata := &Node{Name: "AppData", Parent: marcos, Size: 100, Files: 3}
	windows := &Node{Name: "Windows", Parent: root, Size: 50, Files: 4}
	root.Children = []*Node{users, windows}
	users.Children = []*Node{marcos}
	marcos.Children = []*Node{appdata}
	return root
}

func TestSumTotals(t *testing.T) {
	root := tree()
	SumTotals(root)
	if root.Total != 165 || root.TotalFiles != 10 {
		t.Fatalf("raíz: total=%d archivos=%d", root.Total, root.TotalFiles)
	}
	if u := root.Child("users"); u.Total != 105 {
		t.Fatalf("Users total=%d", u.Total)
	}
}

func TestLookupYPath(t *testing.T) {
	root := tree()
	n := Lookup(root, `C:\users\MARCOS\AppData`)
	if n == nil || n.Name != "AppData" {
		t.Fatalf("Lookup no encontró AppData: %v", n)
	}
	if got := n.Path(); got != `C:\Users\Marcos\AppData` {
		t.Fatalf("Path = %q", got)
	}
	if Lookup(root, `C:\`) != root || Lookup(root, `c:`) != root {
		t.Fatal("Lookup de la raíz")
	}
	for _, p := range []string{`D:\Users`, `C:\Users\Otro`, `C:\Usersx`} {
		if Lookup(root, p) != nil {
			t.Errorf("Lookup(%q) debería ser nil", p)
		}
	}
}

func TestLookupRaizNoUnidad(t *testing.T) {
	root := &Node{Name: `C:\Users`}
	root.Children = []*Node{{Name: "Marcos", Parent: root}}
	if Lookup(root, `C:\Users\Marcos`) == nil {
		t.Fatal("no encontró subcarpeta con raíz no-unidad")
	}
	if Lookup(root, `C:\Users2\Marcos`) != nil {
		t.Fatal("prefijo falso aceptado")
	}
}

func TestJoinPath(t *testing.T) {
	if JoinPath(`C:\`, "a") != `C:\a` || JoinPath(`C:\x`, "a") != `C:\x\a` {
		t.Fatal("JoinPath")
	}
}
