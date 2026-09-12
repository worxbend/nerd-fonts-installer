package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/worxbend/nerd-fonts-installer/internal/config"
	"github.com/worxbend/nerd-fonts-installer/internal/fonts"
	"github.com/worxbend/nerd-fonts-installer/internal/nerdfonts"
	"github.com/worxbend/nerd-fonts-installer/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	errCancelled = errors.New("cancelled")
	errNoConfig  = errors.New("no config found")
)

const (
	commandName  = "nerd-fonts-installer"
	configEnvVar = "NERD_FONTS_INSTALLER_CONFIG"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	os.Exit(run(ctx, os.Args[1:], ioStreams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}, dependencies{}))
}

type ioStreams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type dependencies struct {
	loadConfig         func(string) (config.Config, error)
	discoverConfig     func() (config.Source, bool, error)
	defaultConfigPaths func() ([]string, error)
	listReleases       func(context.Context) ([]nerdfonts.Release, error)
	runTUI             func(context.Context, []nerdfonts.Release, tui.Options) (tui.Result, error)
	installFonts       func(context.Context, fonts.Options) error
	isTerminal         func(ioStreams) bool
}

func (d dependencies) withDefaults() dependencies {
	if d.loadConfig == nil {
		d.loadConfig = config.Load
	}
	if d.discoverConfig == nil {
		d.discoverConfig = config.Discover
	}
	if d.defaultConfigPaths == nil {
		d.defaultConfigPaths = config.DefaultPaths
	}
	if d.listReleases == nil {
		d.listReleases = nerdfonts.Client{}.Releases
	}
	if d.runTUI == nil {
		d.runTUI = tui.Run
	}
	if d.installFonts == nil {
		d.installFonts = fonts.Install
	}
	if d.isTerminal == nil {
		d.isTerminal = isTerminal
	}
	return d
}

func effectiveConfigPath(configPath string, explicit bool) (string, bool) {
	if explicit {
		return configPath, true
	}
	if env := strings.TrimSpace(os.Getenv(configEnvVar)); env != "" {
		return env, true
	}
	return configPath, false
}

func effectiveConfigSource(explicit bool) string {
	if explicit {
		return "flag"
	}
	if env := strings.TrimSpace(os.Getenv(configEnvVar)); env != "" {
		return "env"
	}
	return ""
}

func defaultConfig() config.Config {
	var cfg config.Config
	cfg.ApplyDefaults()
	return cfg
}

type configResolutionError struct {
	path string
	err  error
}

func (e configResolutionError) Error() string {
	return fmt.Sprintf("load config %s: %v", e.path, e.err)
}

func (e configResolutionError) Unwrap() error {
	return e.err
}

func explicitConfigError(path string, err error) error {
	return configResolutionError{path: path, err: err}
}

func exitCodeFor(err error) int {
	var notFound nerdfonts.ReleaseNotFoundError
	var configErr configResolutionError
	switch {
	case errors.As(err, &notFound),
		errors.Is(err, nerdfonts.ErrNoReleases),
		errors.Is(err, errNoConfig),
		errors.As(err, &configErr):
		return 2
	default:
		return 1
	}
}

func noConfigError() error {
	hint := fmt.Sprintf("pass --config or set %s", configEnvVar)
	if paths, err := config.DefaultPaths(); err == nil && len(paths) > 0 {
		hint = fmt.Sprintf("pass --config, set %s, or create one of: %s", configEnvVar, strings.Join(paths, ", "))
	}
	return fmt.Errorf("%w; %s", errNoConfig, hint)
}

func selectRelease(releases []nerdfonts.Release, release string) (nerdfonts.Release, error) {
	if len(releases) == 0 {
		return nerdfonts.Release{}, nerdfonts.ErrNoReleases
	}
	if release == "" || release == nerdfonts.Latest {
		return releases[0], nil
	}
	for _, candidate := range releases {
		if candidate.TagName == release {
			return candidate, nil
		}
	}
	return nerdfonts.Release{}, nerdfonts.ReleaseNotFoundError{Tag: release}
}

func parseIconMode(raw string) (tui.IconMode, error) {
	mode := tui.IconMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case tui.IconAuto, tui.IconNerd, tui.IconUnicode, tui.IconASCII:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --icons %q; use auto, nerd, unicode, or ascii", raw)
	}
}

func install(
	ctx context.Context,
	cfg config.Config,
	opts installOptions,
	deps dependencies,
) error {
	return deps.installFonts(ctx, fonts.Options{
		Release:          cfg.Release,
		Destination:      cfg.Destination,
		Families:         cfg.Families,
		RefreshFontCache: cfg.RefreshFontCache,
		DryRun:           opts.dryRun,
		Stdout:           opts.streams.Out,
		Stderr:           opts.streams.Err,
	})
}

func isTerminal(streams ioStreams) bool {
	stdinFile, stdinOK := streams.In.(*os.File)
	stdoutFile, stdoutOK := streams.Out.(*os.File)
	if !stdinOK || !stdoutOK {
		return false
	}

	stdinInfo, err := stdinFile.Stat()
	if err != nil {
		return false
	}
	stdoutInfo, err := stdoutFile.Stat()
	if err != nil {
		return false
	}
	return stdinInfo.Mode()&os.ModeCharDevice != 0 && stdoutInfo.Mode()&os.ModeCharDevice != 0
}
