package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapAppearancePersistsAndRejectsCredentialFields(t *testing.T) {
	s := setupMapService(t)
	initial, err := s.GetMapAppearance()
	if err != nil || initial.Version != 0 || initial.DarkStyle != "darkblue" {
		t.Fatalf("new installation: %+v, %v", initial, err)
	}
	profile := MapAppearance{Version: 1, DarkStyle: "custom", CustomStyleID: "amap://styles/10146c2104d182971b5311278a987024", ThemeMode: "dark"}
	saved, err := s.SaveMapAppearance(profile)
	if err != nil || saved.CustomStyleID != "10146c2104d182971b5311278a987024" {
		t.Fatalf("saving profile: %+v, %v", saved, err)
	}
	loaded, err := s.GetMapAppearance()
	if err != nil || loaded != saved {
		t.Fatalf("loading profile: %+v, %v", loaded, err)
	}
	data, err := os.ReadFile(ResolveAppDir(appearanceFileName))
	if err != nil || strings.Contains(string(data), "securityJsCode") || strings.Contains(string(data), `"key"`) {
		t.Fatalf("appearance profile contains credentials or is unreadable: %v", err)
	}
	if _, err := s.SaveMapAppearance(MapAppearance{Version: 1, DarkStyle: "custom", CustomStyleID: "", ThemeMode: "dark"}); err == nil {
		t.Fatal("custom selection without ID accepted")
	}
	loaded, err = s.GetMapAppearance()
	if err != nil || loaded != saved {
		t.Fatalf("invalid save changed valid profile: %+v, %v", loaded, err)
	}
	if _, err := decodeMapAppearance([]byte(`{"version":1,"darkStyle":"darkblue","customStyleId":"","themeMode":"system","securityJsCode":"secret"}`)); err == nil {
		t.Fatal("import accepted credential field")
	}
	if _, err := decodeMapAppearance([]byte(`{"version":2,"darkStyle":"darkblue","customStyleId":"","themeMode":"system"}`)); err == nil {
		t.Fatal("import accepted unsupported schema version")
	}
	exportedPath := filepath.Join(t.TempDir(), "appearance.json")
	if err := exportMapAppearanceToPath(exportedPath, saved); err != nil {
		t.Fatal(err)
	}
	imported, err := importMapAppearanceFromPath(exportedPath)
	if err != nil || imported != saved {
		t.Fatalf("export/import round trip: %+v, %v", imported, err)
	}
}
