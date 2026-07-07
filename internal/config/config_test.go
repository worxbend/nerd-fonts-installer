package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fonts.yaml")
	if err := os.WriteFile(path, []byte("families: [JetBrainsMono]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Release != "latest" {
		t.Fatalf("Release = %q", cfg.Release)
	}
	if cfg.Destination != "~/.local/share/fonts/NerdFonts" {
		t.Fatalf("Destination = %q", cfg.Destination)
	}
}

func TestLoadNormalizesValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fonts.yaml")
	data := []byte("release: ' v3.4.0 '\ndestination: ' /tmp/fonts '\nfamilies: [' Hack ', ' JetBrainsMono ']\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Release != "v3.4.0" {
		t.Fatalf("Release = %q, want %q", cfg.Release, "v3.4.0")
	}
	if cfg.Destination != "/tmp/fonts" {
		t.Fatalf("Destination = %q, want %q", cfg.Destination, "/tmp/fonts")
	}
	if got := cfg.Families; len(got) != 2 || got[0] != "Hack" || got[1] != "JetBrainsMono" {
		t.Fatalf("Families = %#v", got)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fonts.yaml")
	data := []byte("families: [Hack]\nfont_family: Hack\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want unknown field error")
	}
}

func TestLoadParsesJSONConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fonts.json")
	data := []byte(`{"release":"v3.4.0","destination":"/tmp/fonts","refresh_font_cache":true,"families":["Hack"]}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Release != "v3.4.0" {
		t.Fatalf("Release = %q, want v3.4.0", cfg.Release)
	}
	if cfg.Destination != "/tmp/fonts" {
		t.Fatalf("Destination = %q, want /tmp/fonts", cfg.Destination)
	}
	if !cfg.RefreshFontCache {
		t.Fatal("RefreshFontCache = false, want true")
	}
	if got := cfg.Families; len(got) != 1 || got[0] != "Hack" {
		t.Fatalf("Families = %#v", got)
	}
}

func TestLoadRejectsBlankAfterTrim(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "release",
			data: "release: '   '\ndestination: /tmp/fonts\nfamilies: [Hack]\n",
		},
		{
			name: "destination",
			data: "release: latest\ndestination: '   '\nfamilies: [Hack]\n",
		},
		{
			name: "family",
			data: "release: latest\ndestination: /tmp/fonts\nfamilies: ['   ']\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fonts.yaml")
			if err := os.WriteFile(path, []byte(tt.data), 0o644); err != nil {
				t.Fatal(err)
			}

			if _, err := Load(path); err == nil {
				t.Fatal("Load() error = nil, want blank value error")
			}
		})
	}
}

func TestValidateRejectsDuplicateFamilies(t *testing.T) {
	cfg := Config{Release: "latest", Destination: "/tmp/fonts", Families: []string{"Hack", "Hack"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want duplicate error")
	}
}

func TestValidateRejectsDuplicateFamiliesAfterTrim(t *testing.T) {
	cfg := Config{Release: "latest", Destination: "/tmp/fonts", Families: []string{"Hack", " Hack "}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want duplicate error")
	}
}

func TestValidateRejectsUnsafeFamilyNames(t *testing.T) {
	tests := []struct {
		name   string
		family string
	}{
		{name: "slash", family: "Hack/Regular"},
		{name: "backslash", family: `Hack\Regular`},
		{name: "absolute", family: "/tmp/Hack"},
		{name: "dot", family: "."},
		{name: "dot dot", family: ".."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Release: "latest", Destination: "/tmp/fonts", Families: []string{tt.family}}
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want unsafe family error")
			}
		})
	}
}

func TestDiscoverPathsUsesFirstExistingConfig(t *testing.T) {
	temp := t.TempDir()
	missing := filepath.Join(temp, "missing.yaml")
	first := filepath.Join(temp, "first.yaml")
	second := filepath.Join(temp, "second.yaml")

	if err := os.WriteFile(first, []byte("families: [Hack]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("families: [JetBrainsMono]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	source, found, err := DiscoverPaths([]string{missing, first, second})
	if err != nil {
		t.Fatalf("DiscoverPaths() error = %v", err)
	}
	if !found {
		t.Fatal("DiscoverPaths() found = false")
	}
	if source.Path != first {
		t.Fatalf("Path = %q, want %q", source.Path, first)
	}
	if got := source.Config.Families; len(got) != 1 || got[0] != "Hack" {
		t.Fatalf("Families = %#v", got)
	}
}

func TestDiscoverPathsReturnsInvalidConfigError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("families: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := DiscoverPaths([]string{path})
	if err == nil {
		t.Fatal("DiscoverPaths() error = nil, want validation error")
	}
}

func TestDefaultPathsSearchesCurrentDirectoryBeforeConfigHome(t *testing.T) {
	temp := t.TempDir()
	cwd := filepath.Join(temp, "cwd")
	configHome := filepath.Join(temp, "xdg")
	home := filepath.Join(temp, "home")
	if err := os.Mkdir(cwd, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}

	wantPrefix := []string{
		filepath.Join(cwd, "nerd-fonts-installer.yaml"),
		filepath.Join(cwd, "nerd-fonts-installer.yml"),
		filepath.Join(cwd, "nerd-fonts-installer.json"),
		filepath.Join(cwd, "nerd-fonts-installer.conf"),
		filepath.Join(cwd, "nerd-fonts-installer", "config.yaml"),
		filepath.Join(cwd, "nerd-fonts-installer", "config.yml"),
		filepath.Join(cwd, "nerd-fonts-installer", "config.json"),
		filepath.Join(cwd, "nerd-fonts-installer", "config.conf"),
		filepath.Join(configHome, "nerd-fonts-installer.yaml"),
		filepath.Join(configHome, "nerd-fonts-installer.yml"),
		filepath.Join(configHome, "nerd-fonts-installer.json"),
		filepath.Join(configHome, "nerd-fonts-installer.conf"),
		filepath.Join(configHome, "nerd-fonts-installer", "config.yaml"),
		filepath.Join(configHome, "nerd-fonts-installer", "config.yml"),
		filepath.Join(configHome, "nerd-fonts-installer", "config.json"),
		filepath.Join(configHome, "nerd-fonts-installer", "config.conf"),
	}
	if len(paths) < len(wantPrefix) {
		t.Fatalf("DefaultPaths() returned %d paths, want at least %d: %#v", len(paths), len(wantPrefix), paths)
	}
	for i, want := range wantPrefix {
		if paths[i] != want {
			t.Fatalf("DefaultPaths()[%d] = %q, want %q; paths = %#v", i, paths[i], want, paths)
		}
	}
}

func TestDefaultPathsFallsBackToHomeConfigWhenXDGUnset(t *testing.T) {
	temp := t.TempDir()
	cwd := filepath.Join(temp, "cwd")
	home := filepath.Join(temp, "home")
	if err := os.Mkdir(cwd, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}

	want := filepath.Join(home, ".config", "nerd-fonts-installer", "config.yaml")
	for _, path := range paths {
		if path == want {
			return
		}
	}
	t.Fatalf("DefaultPaths() = %#v, want path %q", paths, want)
}

func TestDefaultPathsIgnoresRelativeXDGConfigHome(t *testing.T) {
	temp := t.TempDir()
	cwd := filepath.Join(temp, "cwd")
	home := filepath.Join(temp, "home")
	if err := os.Mkdir(cwd, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "relative")

	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}

	want := filepath.Join(home, ".config", "nerd-fonts-installer.yaml")
	for _, path := range paths {
		if path == want {
			return
		}
	}
	t.Fatalf("DefaultPaths() = %#v, want path %q", paths, want)
}

func TestDefaultPathsUsesXDGConfigHome(t *testing.T) {
	cwd := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "xdg")
	t.Chdir(cwd)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}

	want := filepath.Join(configHome, "nerd-fonts-installer", "config.yaml")
	for _, path := range paths {
		if path == want {
			return
		}
	}
	t.Fatalf("DefaultPaths() = %#v, want path %q", paths, want)
}
