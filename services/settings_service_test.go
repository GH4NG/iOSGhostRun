package services

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMapSettingsPersistence(t *testing.T) {
	previous := dataDirectory
	dataDirectory = t.TempDir()
	t.Cleanup(func() { dataDirectory = previous })
	service := NewSettingsService(nil)
	settings, err := service.GetMapSettings()
	if err != nil || settings != (MapSettingsView{}) {
		t.Fatalf("fresh settings = %+v, %v", settings, err)
	}
	want := MapSettingsView{Key: "test-key", SecurityCodeConfigured: true}
	settings, err = service.SaveMapSettings(MapSettings{Key: " test-key ", SecurityJsCode: " test-secret "})
	if err != nil || settings != want {
		t.Fatalf("saved settings = %+v, %v", settings, err)
	}
	settings, err = NewSettingsService(nil).GetMapSettings()
	if err != nil || settings != want {
		t.Fatalf("reloaded settings = %+v, %v", settings, err)
	}
	// Settings responses never disclose the stored security code.
	encoded, _ := json.Marshal(settings)
	if bytes.Contains(encoded, []byte("test-secret")) {
		t.Fatal("settings response leaked security code")
	}
	if _, err := service.SaveMapSettings(MapSettings{Key: "test-key"}); err != nil {
		t.Fatalf("cannot preserve existing secret: %v", err)
	}
	stored, _ := service.readMapSettings()
	if stored.SecurityJsCode != "test-secret" {
		t.Fatal("blank field did not preserve the existing secret")
	}
	if _, err := service.SaveMapSettings(MapSettings{Key: "different-key"}); err == nil {
		t.Fatal("a different key must supply its own secret")
	}
	settings, _ = service.GetMapSettings()
	if settings != want {
		t.Fatal("an invalid save replaced existing settings")
	}
	if _, err := service.SaveMapSettings(MapSettings{}); err != nil {
		t.Fatal(err)
	}
	settings, err = NewSettingsService(nil).GetMapSettings()
	if err != nil || settings != (MapSettingsView{}) {
		t.Fatalf("cleared settings = %+v, %v", settings, err)
	}
	// Old external-proxy configuration keeps the Key, but requires a security code.
	if err := os.WriteFile(ResolveAppDir("map-settings.json"), []byte(`{"key":"legacy-key","serviceHost":"https://example.com/_AMapService"}`), 0600); err != nil {
		t.Fatal(err)
	}
	settings, err = service.GetMapSettings()
	if err != nil || settings.Key != "legacy-key" || settings.SecurityCodeConfigured {
		t.Fatal("legacy configuration was not handled")
	}
	if err := os.WriteFile(filepath.Join(dataDirectory, "map-settings.json"), []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetMapSettings(); err == nil {
		t.Fatal("corrupt settings must report an error")
	}
}

func TestValidateMapSettings(t *testing.T) {
	for _, settings := range []MapSettings{
		{Key: "key"},
		{SecurityJsCode: "secret"},
		{Key: "a b", SecurityJsCode: "secret"},
		{Key: "key", SecurityJsCode: "a b"},
		{Key: "key", SecurityJsCode: "https://example.com/_AMapService"},
		{Key: strings.Repeat("x", 257), SecurityJsCode: "secret"},
	} {
		if _, err := validateMapSettings(settings); err == nil {
			t.Errorf("accepted invalid settings: %+v", settings)
		}
	}
	if _, err := validateMapSettings(MapSettings{Key: "key", SecurityJsCode: "secret"}); err != nil {
		t.Fatal(err)
	}
}
func TestWintunImportValidation(t *testing.T) {
	for arch, folder := range map[string]string{"amd64": "amd64", "arm64": "arm64", "386": "x86", "arm": "arm"} {
		t.Run(arch, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "assets", "wintun", folder, "wintun.dll"))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := readWintunFile("wintun.dll", data, arch); err != nil {
				t.Fatalf("official DLL rejected: %v", err)
			}
			otherArch := "amd64"
			if arch == otherArch {
				otherArch = "arm64"
			}
			if _, _, err := readWintunFile("wintun.dll", data, otherArch); err == nil {
				t.Fatal("accepted DLL for the wrong architecture")
			}
			data[len(data)-1] ^= 1
			if _, _, err := readWintunFile("wintun.dll", data, arch); err == nil || !strings.Contains(err.Error(), "SHA-256") {
				t.Fatalf("tampered DLL should fail hash verification: %v", err)
			}
		})
	}
	for _, filename := range []string{"untrusted.zip", "wintun.dll", "wintun.exe"} {
		if _, _, err := readWintunFile(filename, []byte("not an official runtime"), "amd64"); err == nil {
			t.Errorf("accepted %s", filename)
		}
	}
	if _, _, err := readWintunFile("wintun.zip", make([]byte, maxWintunSize+1), "amd64"); err == nil {
		t.Fatal("accepted oversized archive")
	}
}

func TestWintunOfficialArchive(t *testing.T) {
	path := os.Getenv("IOSGHOSTRUN_TEST_WINTUN_ZIP")
	if path == "" {
		t.Skip("set IOSGHOSTRUN_TEST_WINTUN_ZIP to test a downloaded official archive")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for arch := range wintunDLLHashes {
		if _, license, err := readWintunFile("wintun.zip", data, arch); err != nil || len(license) == 0 {
			t.Fatalf("official archive for %s: missing license or extraction error: %v", arch, err)
		}
	}
	dir := t.TempDir()
	if err := installWintunToDirectory("wintun.zip", data, nil, runtime.GOARCH, dir); err != nil {
		t.Fatal(err)
	}
	dll, _, _ := readWintunFile("wintun.zip", data, runtime.GOARCH)
	installed, err := os.ReadFile(filepath.Join(dir, "wintun.dll"))
	if err != nil || !bytes.Equal(dll, installed) {
		t.Fatal("installed DLL does not match the official archive")
	}
	if _, err := os.Stat(filepath.Join(dir, "WINTUN-LICENSE.txt")); err != nil {
		t.Fatal("license was not installed")
	}
	if err := installWintunToDirectory("wintun.dll", dll, []byte("license"), runtime.GOARCH, dir); err != nil {
		t.Fatalf("reimporting identical DLL should succeed: %v", err)
	}
}
