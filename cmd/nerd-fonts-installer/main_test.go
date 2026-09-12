package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/worxbend/nerd-fonts-installer/internal/config"
	"github.com/worxbend/nerd-fonts-installer/internal/fonts"
	"github.com/worxbend/nerd-fonts-installer/internal/nerdfonts"
	"github.com/worxbend/nerd-fonts-installer/internal/tui"
)

func runCLI(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer, deps dependencies) int {
	return run(ctx, args, ioStreams{
		In:  strings.NewReader(""),
		Out: stdout,
		Err: stderr,
	}, deps)
}

func assertUsageError(t *testing.T, stderr string, wantMessage string, helpCommand string) {
	t.Helper()

	if !strings.Contains(stderr, wantMessage) {
		t.Fatalf("stderr = %q, want substring %q", stderr, wantMessage)
	}

	wantHint := fmt.Sprintf("Run '%s --help' for usage.", helpCommand)
	if !strings.Contains(stderr, wantHint) {
		t.Fatalf("stderr = %q, want help hint %q", stderr, wantHint)
	}

	if strings.Contains(stderr, "Usage:\n") {
		t.Fatalf("stderr = %q, want concise usage error without full help text", stderr)
	}
}

func TestRunVersionFlagAndCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "flag", args: []string{"--version"}, want: []string{"nerd-fonts-installer", "commit:", "date:", "go:", "platform:"}},
		{name: "command", args: []string{"version"}, want: []string{"nerd-fonts-installer", "commit:", "date:", "go:", "platform:"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runCLI(t.Context(), tt.args, &stdout, &stderr, dependencies{})
			if code != 0 {
				t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
			}
			for _, want := range tt.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunVersionJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"version", "--json"}, &stdout, &stderr, dependencies{})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}

	var got map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v; stdout=%q", err, stdout.String())
	}
	for _, key := range []string{"version", "commit", "date", "go_version", "goos", "goarch"} {
		if got[key] == "" {
			t.Fatalf("json output missing %q: %#v", key, got)
		}
	}
}

func TestRunListCommandPlainAndJSON(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStdout string
		wantJSON   bool
	}{
		{name: "plain", args: []string{"list"}, wantStdout: "Hack\nJetBrainsMono\n"},
		{name: "nested noun json", args: []string{"list", "fonts", "--json"}, wantJSON: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runCLI(t.Context(), tt.args, &stdout, &stderr, dependencies{
				discoverConfig: func() (config.Source, bool, error) {
					return config.Source{}, false, nil
				},
				defaultConfigPaths: func() ([]string, error) {
					return []string{"/configs/a.yaml"}, nil
				},
				listReleases: func(context.Context) ([]nerdfonts.Release, error) {
					return []nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack", "JetBrainsMono"}}}, nil
				},
			})
			if code != 0 {
				t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
			}
			if tt.wantJSON {
				var payload listOutput
				if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
					t.Fatalf("json.Unmarshal(stdout) error = %v; stdout=%q", err, stdout.String())
				}
				if payload.Release != "v3.4.0" || len(payload.Families) != 2 || payload.Families[0] != "Hack" {
					t.Fatalf("payload = %#v", payload)
				}
				return
			}
			if stdout.String() != tt.wantStdout {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
		})
	}
}

func TestRunListCommandHonorsReleaseFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"list", "--release", "v3.3.0"}, &stdout, &stderr, dependencies{
		listReleases: func(context.Context) ([]nerdfonts.Release, error) {
			return []nerdfonts.Release{
				{TagName: "v3.4.0", Families: []string{"Hack"}},
				{TagName: "v3.3.0", Families: []string{"FiraCode", "Meslo"}},
			}, nil
		},
		defaultConfigPaths: func() ([]string, error) {
			return []string{"/configs/a.yaml"}, nil
		},
		discoverConfig: func() (config.Source, bool, error) {
			return config.Source{}, false, nil
		},
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if stdout.String() != "FiraCode\nMeslo\n" {
		t.Fatalf("stdout = %q, want selected release families", stdout.String())
	}
}

func TestRunFontNamesBackwardCompatibleAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"--font-names"}, &stdout, &stderr, dependencies{
		discoverConfig: func() (config.Source, bool, error) {
			return config.Source{}, false, nil
		},
		defaultConfigPaths: func() ([]string, error) {
			return []string{"/configs/a.yaml"}, nil
		},
		listReleases: func(context.Context) ([]nerdfonts.Release, error) {
			return []nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack", "JetBrainsMono"}}}, nil
		},
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	want := "# v3.4.0\nfamilies:\n  - Hack\n  - JetBrainsMono\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunInfoCommandWithAndWithoutConfig(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		deps     dependencies
		contains []string
	}{
		{
			name: "with discovered config",
			args: []string{"info"},
			deps: dependencies{
				discoverConfig: func() (config.Source, bool, error) {
					return config.Source{Path: "discovered.yaml", Config: config.Config{
						Release:          "v3.3.0",
						Destination:      "/fonts",
						Families:         []string{"FiraCode", "Meslo"},
						RefreshFontCache: true,
					}}, true, nil
				},
				defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
				listReleases: func(context.Context) ([]nerdfonts.Release, error) {
					return []nerdfonts.Release{{TagName: "v3.3.0", Families: []string{"FiraCode", "Meslo"}}}, nil
				},
			},
			contains: []string{"Config path:", "discovered.yaml", "Config source:", "discovered", "Resolved release:", "v3.3.0", "Families:", "FiraCode, Meslo", "Refresh font cache:", "true"},
		},
		{
			name: "without config uses defaults",
			args: []string{"info", "--json"},
			deps: dependencies{
				discoverConfig:     func() (config.Source, bool, error) { return config.Source{}, false, nil },
				defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
				listReleases: func(context.Context) ([]nerdfonts.Release, error) {
					return []nerdfonts.Release{{TagName: "v3.5.0", Families: []string{"Hack"}}}, nil
				},
			},
			contains: []string{"\"config_path\": \"none found\"", "\"config_source\": \"none\"", "\"release_selector\": \"latest\"", "\"resolved_release\": \"v3.5.0\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runCLI(t.Context(), tt.args, &stdout, &stderr, tt.deps)
			if code != 0 {
				t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
			}
			for _, want := range tt.contains {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout = %q, want substring %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestRunHelpToStdoutIncludesSubcommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"--help"}, &stdout, &stderr, dependencies{})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	for _, want := range []string{"install", "list", "info", "version", "completion", "Examples:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want substring %q", stdout.String(), want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunSubcommandHelpToStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"install", "--help"}, &stdout, &stderr, dependencies{})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"Usage:", commandName + " install [flags]", "--dry-run"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want substring %q", stdout.String(), want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReturnsUsageCodeForNewErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		deps       dependencies
		wantCode   int
		wantStderr string
		helpCmd    string
	}{
		{name: "unknown command", args: []string{"bogus"}, wantCode: 2, wantStderr: "unknown command \"bogus\"", helpCmd: commandName},
		{name: "list bad target", args: []string{"list", "bogus"}, wantCode: 2, wantStderr: "unknown list target \"bogus\"", helpCmd: commandName + " list"},
		{name: "list missing release", args: []string{"list", "--release", "v1.0.0"}, deps: dependencies{
			listReleases: func(context.Context) ([]nerdfonts.Release, error) {
				return []nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack"}}}, nil
			},
			defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
			discoverConfig:     func() (config.Source, bool, error) { return config.Source{}, false, nil },
		}, wantCode: 2, wantStderr: `release "v1.0.0" was not found`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runCLI(t.Context(), tt.args, &stdout, &stderr, tt.deps)
			if code != tt.wantCode {
				t.Fatalf("run() code = %d, want %d; stderr=%q", code, tt.wantCode, stderr.String())
			}
			if tt.helpCmd != "" {
				assertUsageError(t, stderr.String(), tt.wantStderr, tt.helpCmd)
				return
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRunUsageErrorsPrintHintWithoutFullHelp(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMessage string
		helpCmd     string
	}{
		{
			name:        "root parse error",
			args:        []string{"--bogus"},
			wantMessage: "flag provided but not defined: -bogus",
			helpCmd:     commandName,
		},
		{
			name:        "list parse error",
			args:        []string{"list", "--bogus"},
			wantMessage: "flag provided but not defined: -bogus",
			helpCmd:     commandName + " list",
		},
		{
			name:        "completion unsupported shell",
			args:        []string{"completion", "fish"},
			wantMessage: `unsupported shell "fish"; use bash or zsh`,
			helpCmd:     commandName + " completion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := runCLI(t.Context(), tt.args, &stdout, &stderr, dependencies{})
			if code != 2 {
				t.Fatalf("run() code = %d, want 2; stderr=%q", code, stderr.String())
			}

			assertUsageError(t, stderr.String(), tt.wantMessage, tt.helpCmd)
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunLoadsConfigFromEnvOverride(t *testing.T) {
	t.Setenv(configEnvVar, "/env/fonts.yaml")
	var stderr bytes.Buffer
	var gotPath string
	installed := false

	code := runCLI(t.Context(), nil, &bytes.Buffer{}, &stderr, dependencies{
		loadConfig: func(path string) (config.Config, error) {
			gotPath = path
			return config.Config{Release: "v3.4.0", Destination: "/fonts", Families: []string{"Hack"}}, nil
		},
		discoverConfig: func() (config.Source, bool, error) {
			t.Fatal("discovery must not run when the env override is set")
			return config.Source{}, false, nil
		},
		installFonts: func(context.Context, fonts.Options) error {
			installed = true
			return nil
		},
		isTerminal: func(ioStreams) bool { return false },
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if gotPath != "/env/fonts.yaml" {
		t.Fatalf("loaded path = %q, want the env override", gotPath)
	}
	if !installed {
		t.Fatal("install should have run with the env config")
	}
}

func TestRunLoadsExplicitConfigAndInstalls(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var got fonts.Options

	code := runCLI(t.Context(), []string{"install", "--config", "fonts.yaml", "--dry-run"}, &stdout, &stderr, dependencies{
		loadConfig: func(path string) (config.Config, error) {
			if path != "fonts.yaml" {
				t.Fatalf("load path = %q, want fonts.yaml", path)
			}
			return config.Config{
				Release:          "v3.4.0",
				Destination:      "/fonts",
				Families:         []string{"Hack"},
				RefreshFontCache: true,
			}, nil
		},
		installFonts: func(ctx context.Context, opts fonts.Options) error {
			got = opts
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got.Release != "v3.4.0" || got.Destination != "/fonts" || !got.DryRun || len(got.Families) != 1 || got.Families[0] != "Hack" {
		t.Fatalf("install options = %#v", got)
	}
	if got.Stdout != &stdout || got.Stderr != &stderr {
		t.Fatal("install writers were not passed through")
	}
}

func TestRunReportsExplicitConfigLoadError(t *testing.T) {
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"info", "--config", "missing.yaml"}, &bytes.Buffer{}, &stderr, dependencies{
		loadConfig: func(string) (config.Config, error) {
			return config.Config{}, errors.New("missing")
		},
	})
	if code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "load config missing.yaml") {
		t.Fatalf("stderr = %q, want load context", stderr.String())
	}
}

func TestRunUsesDiscoveredConfig(t *testing.T) {
	var stderr bytes.Buffer
	installed := false

	code := runCLI(t.Context(), nil, &bytes.Buffer{}, &stderr, dependencies{
		discoverConfig: func() (config.Source, bool, error) {
			return config.Source{
				Path: "discovered.yaml",
				Config: config.Config{
					Release:     "latest",
					Destination: "/fonts",
					Families:    []string{"Hack"},
				},
			}, true, nil
		},
		defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
		installFonts: func(context.Context, fonts.Options) error {
			installed = true
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !installed {
		t.Fatal("install was not called")
	}
	if !strings.Contains(stderr.String(), "Using config discovered.yaml") {
		t.Fatalf("stderr = %q, want discovery message", stderr.String())
	}
}

func TestRunInteractiveCancellationIsSuccess(t *testing.T) {
	var stderr bytes.Buffer
	var gotIcons tui.IconMode
	defaults := defaultConfig()

	code := runCLI(t.Context(), []string{"install", "--interactive", "--icons", "nerd"}, &bytes.Buffer{}, &stderr, dependencies{
		discoverConfig:     func() (config.Source, bool, error) { return config.Source{}, false, nil },
		defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
		isTerminal:         func(ioStreams) bool { return true },
		listReleases: func(context.Context) ([]nerdfonts.Release, error) {
			return []nerdfonts.Release{{Name: "v3.4.0", TagName: "v3.4.0", Families: []string{"Hack"}}}, nil
		},
		runTUI: func(_ context.Context, _ []nerdfonts.Release, opts tui.Options) (tui.Result, error) {
			gotIcons = opts.Icons
			if opts.RefreshFontCache != defaults.RefreshFontCache {
				t.Fatalf("TUI refresh_font_cache = %t, want default %t", opts.RefreshFontCache, defaults.RefreshFontCache)
			}
			return tui.Result{Cancelled: true}, nil
		},
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if gotIcons != tui.IconNerd {
		t.Fatalf("TUI icons = %q, want %q", gotIcons, tui.IconNerd)
	}
}

func TestRunInstallRejectsUnsupportedRootFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "release before install",
			args:       []string{"--release", "v3.4.0", "install", "--config", "fonts.yaml"},
			wantStderr: "--release is not supported for install; use --config to select a release",
		},
		{
			name:       "json before install",
			args:       []string{"--json", "install", "--config", "fonts.yaml"},
			wantStderr: "--json is not supported for install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runCLI(t.Context(), tt.args, &stdout, &stderr, dependencies{})
			if code != 2 {
				t.Fatalf("run() code = %d, want 2; stderr=%q", code, stderr.String())
			}
			assertUsageError(t, stderr.String(), tt.wantStderr, commandName+" install")
		})
	}
}

func TestRunExplicitConfigErrorsUseUsageExitCode(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		envValue   string
		loadErr    error
		wantStderr string
	}{
		{
			name:       "install explicit missing config",
			args:       []string{"install", "--config", "missing.yaml"},
			loadErr:    errors.New("missing"),
			wantStderr: "load config missing.yaml: missing",
		},
		{
			name:       "env missing config",
			envValue:   "/env/missing.yaml",
			loadErr:    errors.New("missing"),
			wantStderr: "load config /env/missing.yaml: missing",
		},
		{
			name:       "env invalid config",
			args:       []string{"info"},
			envValue:   "/env/invalid.yaml",
			loadErr:    errors.New("parse /env/invalid.yaml: invalid yaml"),
			wantStderr: "load config /env/invalid.yaml: parse /env/invalid.yaml: invalid yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(configEnvVar, tt.envValue)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runCLI(t.Context(), tt.args, &stdout, &stderr, dependencies{
				loadConfig: func(string) (config.Config, error) {
					return config.Config{}, tt.loadErr
				},
				discoverConfig: func() (config.Source, bool, error) {
					t.Fatal("discovery must not run for explicit config resolution")
					return config.Source{}, false, nil
				},
			})
			if code != 2 {
				t.Fatalf("run() code = %d, want 2; stderr=%q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestInteractiveDefaultsUseConfigApplyDefaults(t *testing.T) {
	defaults := defaultConfig()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "install interactive passes default refresh setting to tui",
			run: func(t *testing.T) {
				code := runCLI(t.Context(), []string{"install", "--interactive"}, &bytes.Buffer{}, &bytes.Buffer{}, dependencies{
					discoverConfig:     func() (config.Source, bool, error) { return config.Source{}, false, nil },
					defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
					isTerminal:         func(ioStreams) bool { return true },
					listReleases: func(context.Context) ([]nerdfonts.Release, error) {
						return []nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack"}}}, nil
					},
					runTUI: func(_ context.Context, _ []nerdfonts.Release, opts tui.Options) (tui.Result, error) {
						if opts.Destination != defaults.Destination {
							t.Fatalf("TUI destination = %q, want %q", opts.Destination, defaults.Destination)
						}
						if opts.RefreshFontCache != defaults.RefreshFontCache {
							t.Fatalf("TUI refresh_font_cache = %t, want default %t", opts.RefreshFontCache, defaults.RefreshFontCache)
						}
						return tui.Result{Cancelled: true}, nil
					},
				})
				if code != 0 {
					t.Fatalf("run() code = %d, want 0", code)
				}
			},
		},
		{
			name: "info interactive reports default refresh setting",
			run: func(t *testing.T) {
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				code := runCLI(t.Context(), []string{"info", "--interactive", "--json"}, &stdout, &stderr, dependencies{
					discoverConfig:     func() (config.Source, bool, error) { return config.Source{}, false, nil },
					defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml"}, nil },
					listReleases: func(context.Context) ([]nerdfonts.Release, error) {
						return []nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack"}}}, nil
					},
				})
				if code != 0 {
					t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
				}
				var payload infoOutput
				if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
					t.Fatalf("json.Unmarshal(stdout) error = %v; stdout=%q", err, stdout.String())
				}
				if payload.ConfigSource != "interactive" {
					t.Fatalf("config source = %q, want interactive", payload.ConfigSource)
				}
				if payload.RefreshFontCache != defaults.RefreshFontCache {
					t.Fatalf("refresh_font_cache = %t, want default %t", payload.RefreshFontCache, defaults.RefreshFontCache)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestRunVerboseWritesDiagnosticsToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI(t.Context(), []string{"list", "--verbose"}, &stdout, &stderr, dependencies{
		discoverConfig:     func() (config.Source, bool, error) { return config.Source{}, false, nil },
		defaultConfigPaths: func() ([]string, error) { return []string{"/configs/a.yaml", "/configs/b.yaml"}, nil },
		listReleases: func(context.Context) ([]nerdfonts.Release, error) {
			return []nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack"}}}, nil
		},
	})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"verbose: checking config candidate /configs/a.yaml", "verbose: no config discovered", "verbose: listing Nerd Fonts releases"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, want substring %q", stderr.String(), want)
		}
	}
}

func TestEffectiveConfigPathFlagWins(t *testing.T) {
	t.Setenv(configEnvVar, "/env/fonts.yaml")
	got, explicit := effectiveConfigPath("/flag/fonts.yaml", true)
	if got != "/flag/fonts.yaml" || !explicit {
		t.Fatalf("effectiveConfigPath = %q, %v; want /flag/fonts.yaml, true", got, explicit)
	}
}

func TestEffectiveConfigPathEnvOverride(t *testing.T) {
	t.Setenv(configEnvVar, "/env/fonts.yaml")
	got, explicit := effectiveConfigPath("", false)
	if got != "/env/fonts.yaml" || !explicit {
		t.Fatalf("effectiveConfigPath = %q, %v; want /env/fonts.yaml, true", got, explicit)
	}
}

func TestEffectiveConfigPathWhitespaceOnlyEnvFallsThrough(t *testing.T) {
	t.Setenv(configEnvVar, "   ")
	got, explicit := effectiveConfigPath("/fallback", false)
	if got != "/fallback" || explicit {
		t.Fatalf("effectiveConfigPath = %q, %v; want /fallback, false", got, explicit)
	}
}

func TestExitCodeForReleaseNotFound(t *testing.T) {
	err := nerdfonts.ReleaseNotFoundError{Tag: "v9.9.9"}
	if got := exitCodeFor(err); got != 2 {
		t.Fatalf("exitCodeFor(ReleaseNotFoundError) = %d, want 2", got)
	}
}

func TestExitCodeForWrappedReleaseNotFound(t *testing.T) {
	wrapped := errors.Join(errors.New("outer"), nerdfonts.ReleaseNotFoundError{Tag: "v9.9.9"})
	if got := exitCodeFor(wrapped); got != 2 {
		t.Fatalf("exitCodeFor(wrapped ReleaseNotFoundError) = %d, want 2", got)
	}
}

func TestExitCodeForNoReleases(t *testing.T) {
	if got := exitCodeFor(nerdfonts.ErrNoReleases); got != 2 {
		t.Fatalf("exitCodeFor(ErrNoReleases) = %d, want 2", got)
	}
}

func TestExitCodeForNoConfig(t *testing.T) {
	err := noConfigError()
	if got := exitCodeFor(err); got != 2 {
		t.Fatalf("exitCodeFor(noConfigError()) = %d, want 2", got)
	}
}

func TestExitCodeForExplicitConfigError(t *testing.T) {
	err := explicitConfigError("missing.yaml", errors.New("missing"))
	if got := exitCodeFor(err); got != 2 {
		t.Fatalf("exitCodeFor(explicitConfigError) = %d, want 2", got)
	}
}

func TestExitCodeForRuntimeError(t *testing.T) {
	if got := exitCodeFor(errors.New("network failure")); got != 1 {
		t.Fatalf("exitCodeFor(generic error) = %d, want 1", got)
	}
}

func TestNoConfigErrorWrapsErrNoConfig(t *testing.T) {
	err := noConfigError()
	if !errors.Is(err, errNoConfig) {
		t.Fatalf("noConfigError() does not wrap errNoConfig: %v", err)
	}
}

func TestNoConfigErrorMentionsEnvVarAndFlag(t *testing.T) {
	err := noConfigError()
	if !strings.Contains(err.Error(), configEnvVar) || !strings.Contains(err.Error(), "--config") {
		t.Fatalf("noConfigError() = %q, want mentions of %s and --config", err.Error(), configEnvVar)
	}
}

func TestSelectRelease(t *testing.T) {
	releases := []nerdfonts.Release{{TagName: "v3.4.0"}, {TagName: "v3.3.0"}}
	if got, err := selectRelease(releases, nerdfonts.Latest); err != nil || got.TagName != "v3.4.0" {
		t.Fatalf("selectRelease(latest) = %q, %v; want v3.4.0, nil", got.TagName, err)
	}
	if got, err := selectRelease(releases, "v3.3.0"); err != nil || got.TagName != "v3.3.0" {
		t.Fatalf("selectRelease(v3.3.0) = %q, %v; want v3.3.0, nil", got.TagName, err)
	}
	if _, err := selectRelease(nil, nerdfonts.Latest); !errors.Is(err, nerdfonts.ErrNoReleases) {
		t.Fatalf("selectRelease(nil) error = %v, want ErrNoReleases", err)
	}
}
