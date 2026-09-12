package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/worxbend/nerd-fonts-installer/internal/config"
	"github.com/worxbend/nerd-fonts-installer/internal/nerdfonts"
	"github.com/worxbend/nerd-fonts-installer/internal/tui"
)

type rootOptions struct {
	configPath     string
	configExplicit bool
	dryRun         bool
	showFontNames  bool
	interactive    bool
	iconMode       string
	showVersion    bool
	release        string
	json           bool
	verbose        bool
	command        string
	commandArgs    []string
}

type commandInvocation struct {
	streams ioStreams
	root    rootOptions
	args    []string
}

type installOptions struct {
	configPath     string
	configExplicit bool
	dryRun         bool
	interactive    bool
	iconMode       string
	verbose        bool
	streams        ioStreams
}

type listOptions struct {
	configPath     string
	configExplicit bool
	release        string
	json           bool
	verbose        bool
}

type infoOptions struct {
	configPath     string
	configExplicit bool
	interactive    bool
	json           bool
	verbose        bool
}

type versionOptions struct {
	json bool
}

type completionOptions struct {
	shell string
}

type releaseLookupOptions struct {
	release string
	verbose bool
}

var (
	rootHelp = helpTopic{
		printHelp: printRootHelp,
	}
	installHelp = helpTopic{
		command:   "install",
		printHelp: printInstallHelp,
	}
	listHelp = helpTopic{
		command:   "list",
		printHelp: printListHelp,
	}
	infoHelp = helpTopic{
		command:   "info",
		printHelp: printInfoHelp,
	}
	versionHelp = helpTopic{
		command:   "version",
		printHelp: printVersionHelp,
	}
	completionHelp = helpTopic{
		command:   "completion",
		printHelp: printCompletionHelp,
	}
)

type configSummary struct {
	path   string
	source string
	cfg    config.Config
}

type listOutput struct {
	Release  string   `json:"release"`
	Families []string `json:"families"`
}

type infoOutput struct {
	ConfigPath       string   `json:"config_path"`
	ConfigSource     string   `json:"config_source"`
	ReleaseSelector  string   `json:"release_selector"`
	ResolvedRelease  string   `json:"resolved_release"`
	Destination      string   `json:"destination"`
	Families         []string `json:"families"`
	RefreshFontCache bool     `json:"refresh_font_cache"`
	GoOS             string   `json:"goos"`
	GoArch           string   `json:"goarch"`
	GoVersion        string   `json:"go_version"`
	Version          string   `json:"version"`
	Commit           string   `json:"commit"`
	Date             string   `json:"date"`
}

type versionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	GoOS      string `json:"goos"`
	GoArch    string `json:"goarch"`
}

func run(
	ctx context.Context,
	args []string,
	streams ioStreams,
	deps dependencies,
) int {
	deps = deps.withDefaults()

	root, err := parseRootFlags(args)
	if err != nil {
		return handleParseError(err, streams, rootHelp)
	}

	if root.showVersion && root.showFontNames {
		return usageError(streams, rootHelp, "--version and --font-names cannot be used together")
	}

	if root.showVersion {
		if root.command != "" {
			return usageError(streams, rootHelp, "--version does not accept a subcommand")
		}
		return runVersion(streams.Out, versionOptions{json: root.json})
	}

	if root.showFontNames {
		if root.command != "" {
			return usageError(streams, rootHelp, "--font-names does not accept a subcommand")
		}
		return runLegacyFontNames(ctx, commandInvocation{streams: streams, root: root}, deps)
	}

	if root.command != "" {
		invocation := commandInvocation{
			streams: streams,
			root:    root,
			args:    root.commandArgs,
		}

		switch root.command {
		case "install":
			return runInstall(ctx, invocation, deps)
		case "list":
			return runList(ctx, invocation, deps)
		case "info":
			return runInfo(ctx, invocation, deps)
		case "version":
			return runVersionCommand(invocation)
		case "completion":
			return runCompletionCommand(invocation)
		default:
			return usageError(streams, rootHelp, fmt.Sprintf("unknown command %q", root.command))
		}
	}

	if strings.TrimSpace(root.release) != "" {
		return usageError(streams, rootHelp, "--release requires the list command or --font-names")
	}
	if root.json {
		return usageError(streams, rootHelp, "--json requires the list, info, or version command")
	}
	return runInstallDefault(ctx, streams, root, deps)
}

func parseRootFlags(args []string) (rootOptions, error) {
	flags := flag.NewFlagSet(commandName, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var opts rootOptions
	flags.StringVar(&opts.configPath, "config", "", "config file path")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "print planned downloads without installing fonts")
	flags.BoolVar(&opts.showFontNames, "font-names", false, "deprecated: use 'list' to print YAML-ready Nerd Font family names and exit")
	flags.BoolVar(&opts.interactive, "interactive", false, "start the terminal picker when no config file is found")
	flags.StringVar(&opts.iconMode, "icons", string(tui.IconAuto), "interactive icon mode: auto, nerd, unicode, or ascii")
	flags.BoolVar(&opts.showVersion, "version", false, "print version information and exit")
	flags.StringVar(&opts.release, "release", "", "release tag for list or --font-names")
	flags.BoolVar(&opts.json, "json", false, "print machine-readable JSON")
	flags.BoolVar(&opts.verbose, "verbose", false, "print extra diagnostics to stderr")
	flags.BoolVar(&opts.verbose, "v", false, "print extra diagnostics to stderr")

	if err := flags.Parse(args); err != nil {
		return rootOptions{}, err
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			opts.configExplicit = true
		}
	})

	remaining := flags.Args()
	if len(remaining) == 0 {
		return opts, nil
	}

	opts.command = remaining[0]
	opts.commandArgs = append([]string{}, remaining[1:]...)
	return opts, nil
}

func runInstallDefault(
	ctx context.Context,
	streams ioStreams,
	root rootOptions,
	deps dependencies,
) int {
	if _, err := parseIconMode(root.iconMode); err != nil {
		return usageError(streams, rootHelp, err.Error())
	}

	opts := installOptions{
		configPath:     root.configPath,
		configExplicit: root.configExplicit,
		dryRun:         root.dryRun,
		interactive:    root.interactive,
		iconMode:       root.iconMode,
		verbose:        root.verbose,
		streams:        streams,
	}
	cfg, err := resolveConfig(ctx, streams, opts, deps)
	if err != nil {
		if errors.Is(err, errCancelled) {
			return 0
		}
		_, _ = fmt.Fprintf(streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}

	if err := install(ctx, cfg, opts, deps); err != nil {
		_, _ = fmt.Fprintf(streams.Err, "install fonts: %v\n", err)
		return 1
	}
	return 0
}

func runInstall(ctx context.Context, invocation commandInvocation, deps dependencies) int {
	if release := strings.TrimSpace(invocation.root.release); release != "" {
		return usageError(invocation.streams, installHelp, "--release is not supported for install; use --config to select a release")
	}
	if invocation.root.json {
		return usageError(invocation.streams, installHelp, "--json is not supported for install")
	}

	flags := flag.NewFlagSet(commandName+" install", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	opts := installOptions{
		configPath:  invocation.root.configPath,
		dryRun:      invocation.root.dryRun,
		interactive: invocation.root.interactive,
		iconMode:    invocation.root.iconMode,
		verbose:     invocation.root.verbose,
		streams:     invocation.streams,
	}
	flags.StringVar(&opts.configPath, "config", opts.configPath, "config file path")
	flags.BoolVar(&opts.dryRun, "dry-run", opts.dryRun, "print planned downloads without installing fonts")
	flags.BoolVar(&opts.interactive, "interactive", opts.interactive, "start the terminal picker when no config file is found")
	flags.StringVar(&opts.iconMode, "icons", opts.iconMode, "interactive icon mode: auto, nerd, unicode, or ascii")
	flags.BoolVar(&opts.verbose, "verbose", opts.verbose, "print extra diagnostics to stderr")
	flags.BoolVar(&opts.verbose, "v", opts.verbose, "print extra diagnostics to stderr")

	if err := flags.Parse(invocation.args); err != nil {
		return handleParseError(err, invocation.streams, installHelp)
	}
	if len(flags.Args()) > 0 {
		return usageError(
			invocation.streams,
			installHelp,
			fmt.Sprintf("install does not accept arguments: %s", strings.Join(flags.Args(), " ")),
		)
	}

	opts.configExplicit = invocation.root.configExplicit
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			opts.configExplicit = true
		}
	})

	if _, err := parseIconMode(opts.iconMode); err != nil {
		return usageError(invocation.streams, installHelp, err.Error())
	}

	cfg, err := resolveConfig(ctx, invocation.streams, opts, deps)
	if err != nil {
		if errors.Is(err, errCancelled) {
			return 0
		}
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}

	if err := install(ctx, cfg, opts, deps); err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "install fonts: %v\n", err)
		return 1
	}
	return 0
}

func runLegacyFontNames(ctx context.Context, invocation commandInvocation, deps dependencies) int {
	opts := listOptions{
		configPath:     invocation.root.configPath,
		configExplicit: invocation.root.configExplicit,
		release:        invocation.root.release,
		json:           invocation.root.json,
		verbose:        invocation.root.verbose,
	}

	release, err := resolveListRelease(invocation.streams, opts, deps)
	if err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}
	selected, err := loadRelease(ctx, invocation.streams, releaseLookupOptions{
		release: release,
		verbose: opts.verbose,
	}, deps)
	if err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}
	if opts.json {
		return writeJSON(invocation.streams.Out, listOutput{
			Release:  selected.TagName,
			Families: append([]string{}, selected.Families...),
		}, invocation.streams.Err)
	}
	printLegacyFontNames(invocation.streams.Out, selected)
	return 0
}

func runList(ctx context.Context, invocation commandInvocation, deps dependencies) int {
	args := append([]string{}, invocation.args...)
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if args[0] != "fonts" {
			return usageError(invocation.streams, listHelp, fmt.Sprintf("unknown list target %q", args[0]))
		}
		args = args[1:]
	}

	flags := flag.NewFlagSet(commandName+" list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	opts := listOptions{
		configPath: invocation.root.configPath,
		release:    invocation.root.release,
		json:       invocation.root.json,
		verbose:    invocation.root.verbose,
	}
	flags.StringVar(&opts.configPath, "config", opts.configPath, "config file path")
	flags.StringVar(&opts.release, "release", opts.release, "release tag to inspect")
	flags.BoolVar(&opts.json, "json", opts.json, "print machine-readable JSON")
	flags.BoolVar(&opts.verbose, "verbose", opts.verbose, "print extra diagnostics to stderr")
	flags.BoolVar(&opts.verbose, "v", opts.verbose, "print extra diagnostics to stderr")

	if err := flags.Parse(args); err != nil {
		return handleParseError(err, invocation.streams, listHelp)
	}
	if len(flags.Args()) > 0 {
		return usageError(
			invocation.streams,
			listHelp,
			fmt.Sprintf("list does not accept arguments: %s", strings.Join(flags.Args(), " ")),
		)
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			opts.configExplicit = true
		}
	})
	opts.configExplicit = opts.configExplicit || invocation.root.configExplicit

	release, err := resolveListRelease(invocation.streams, opts, deps)
	if err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}
	selected, err := loadRelease(ctx, invocation.streams, releaseLookupOptions{
		release: release,
		verbose: opts.verbose,
	}, deps)
	if err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}

	if opts.json {
		return writeJSON(invocation.streams.Out, listOutput{
			Release:  selected.TagName,
			Families: append([]string{}, selected.Families...),
		}, invocation.streams.Err)
	}
	for _, family := range selected.Families {
		_, _ = fmt.Fprintln(invocation.streams.Out, family)
	}
	return 0
}

func runInfo(ctx context.Context, invocation commandInvocation, deps dependencies) int {
	flags := flag.NewFlagSet(commandName+" info", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	opts := infoOptions{
		configPath:  invocation.root.configPath,
		interactive: invocation.root.interactive,
		json:        invocation.root.json,
		verbose:     invocation.root.verbose,
	}
	flags.StringVar(&opts.configPath, "config", opts.configPath, "config file path")
	flags.BoolVar(&opts.interactive, "interactive", opts.interactive, "report interactive defaults when no config file is found")
	flags.BoolVar(&opts.json, "json", opts.json, "print machine-readable JSON")
	flags.BoolVar(&opts.verbose, "verbose", opts.verbose, "print extra diagnostics to stderr")
	flags.BoolVar(&opts.verbose, "v", opts.verbose, "print extra diagnostics to stderr")

	if err := flags.Parse(invocation.args); err != nil {
		return handleParseError(err, invocation.streams, infoHelp)
	}
	if len(flags.Args()) > 0 {
		return usageError(
			invocation.streams,
			infoHelp,
			fmt.Sprintf("info does not accept arguments: %s", strings.Join(flags.Args(), " ")),
		)
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			opts.configExplicit = true
		}
	})
	opts.configExplicit = opts.configExplicit || invocation.root.configExplicit

	summary, err := inspectConfig(invocation.streams, opts, deps)
	if err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}

	selected, err := loadRelease(ctx, invocation.streams, releaseLookupOptions{
		release: summary.cfg.Release,
		verbose: opts.verbose,
	}, deps)
	if err != nil {
		_, _ = fmt.Fprintf(invocation.streams.Err, "%v\n", err)
		return exitCodeFor(err)
	}

	payload := infoOutput{
		ConfigPath:       displayConfigPath(summary.path),
		ConfigSource:     summary.source,
		ReleaseSelector:  summary.cfg.Release,
		ResolvedRelease:  selected.TagName,
		Destination:      summary.cfg.Destination,
		Families:         append([]string{}, summary.cfg.Families...),
		RefreshFontCache: summary.cfg.RefreshFontCache,
		GoOS:             runtime.GOOS,
		GoArch:           runtime.GOARCH,
		GoVersion:        runtime.Version(),
		Version:          version,
		Commit:           commit,
		Date:             date,
	}
	if opts.json {
		return writeJSON(invocation.streams.Out, payload, invocation.streams.Err)
	}
	printInfo(invocation.streams.Out, payload)
	return 0
}

func runVersionCommand(invocation commandInvocation) int {
	flags := flag.NewFlagSet(commandName+" version", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	opts := versionOptions{json: invocation.root.json}
	flags.BoolVar(&opts.json, "json", opts.json, "print machine-readable JSON")
	if err := flags.Parse(invocation.args); err != nil {
		return handleParseError(err, invocation.streams, versionHelp)
	}
	if len(flags.Args()) > 0 {
		return usageError(
			invocation.streams,
			versionHelp,
			fmt.Sprintf("version does not accept arguments: %s", strings.Join(flags.Args(), " ")),
		)
	}
	return runVersion(invocation.streams.Out, opts)
}

func runVersion(stdout io.Writer, opts versionOptions) int {
	payload := versionOutput{
		Version:   version,
		Commit:    commit,
		Date:      date,
		GoVersion: runtime.Version(),
		GoOS:      runtime.GOOS,
		GoArch:    runtime.GOARCH,
	}
	if opts.json {
		return writeJSON(stdout, payload, io.Discard)
	}
	_, _ = fmt.Fprintf(stdout, "%s %s\n", commandName, payload.Version)
	_, _ = fmt.Fprintf(stdout, "commit: %s\n", payload.Commit)
	_, _ = fmt.Fprintf(stdout, "date: %s\n", payload.Date)
	_, _ = fmt.Fprintf(stdout, "go: %s\n", payload.GoVersion)
	_, _ = fmt.Fprintf(stdout, "platform: %s/%s\n", payload.GoOS, payload.GoArch)
	return 0
}

func runCompletionCommand(invocation commandInvocation) int {
	flags := flag.NewFlagSet(commandName+" completion", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(invocation.args); err != nil {
		return handleParseError(err, invocation.streams, completionHelp)
	}
	remaining := flags.Args()
	if len(remaining) != 1 {
		return usageError(invocation.streams, completionHelp, "completion requires exactly one shell: bash or zsh")
	}
	opts := completionOptions{shell: remaining[0]}
	script, err := completionScript(opts.shell)
	if err != nil {
		return usageError(invocation.streams, completionHelp, err.Error())
	}
	_, _ = io.WriteString(invocation.streams.Out, script)
	return 0
}

func resolveListRelease(streams ioStreams, opts listOptions, deps dependencies) (string, error) {
	if release := strings.TrimSpace(opts.release); release != "" {
		verbosef(opts.verbose, streams.Err, "using explicit release selector %q", release)
		return release, nil
	}
	if path, explicit := effectiveConfigPath(opts.configPath, opts.configExplicit); explicit {
		source := effectiveConfigSource(opts.configExplicit)
		verbosef(opts.verbose, streams.Err, "loading release selector from %s config %s", source, path)
		cfg, err := deps.loadConfig(path)
		if err != nil {
			return "", explicitConfigError(path, err)
		}
		return cfg.Release, nil
	}
	source, found, err := discoverConfig(opts.verbose, streams.Err, deps)
	if err != nil {
		return "", err
	}
	if found {
		return source.Config.Release, nil
	}
	verbosef(opts.verbose, streams.Err, "no config found; defaulting release selector to %q", nerdfonts.Latest)
	return nerdfonts.Latest, nil
}

func loadRelease(ctx context.Context, streams ioStreams, opts releaseLookupOptions, deps dependencies) (nerdfonts.Release, error) {
	verbosef(opts.verbose, streams.Err, "listing Nerd Fonts releases")
	releases, err := deps.listReleases(ctx)
	if err != nil {
		return nerdfonts.Release{}, err
	}
	selected, err := selectRelease(releases, opts.release)
	if err != nil {
		return nerdfonts.Release{}, err
	}
	verbosef(opts.verbose, streams.Err, "resolved release selector %q to %q", opts.release, selected.TagName)
	return selected, nil
}

func inspectConfig(streams ioStreams, opts infoOptions, deps dependencies) (configSummary, error) {
	if path, explicit := effectiveConfigPath(opts.configPath, opts.configExplicit); explicit {
		source := effectiveConfigSource(opts.configExplicit)
		verbosef(opts.verbose, streams.Err, "loading config from %s path %s", source, path)
		cfg, err := deps.loadConfig(path)
		if err != nil {
			return configSummary{}, explicitConfigError(path, err)
		}
		return configSummary{
			path:   path,
			source: source,
			cfg:    cfg,
		}, nil
	}

	source, found, err := discoverConfig(opts.verbose, streams.Err, deps)
	if err != nil {
		return configSummary{}, err
	}
	if found {
		return configSummary{
			path:   source.Path,
			source: "discovered",
			cfg:    source.Config,
		}, nil
	}

	cfg := defaultConfig()
	if opts.interactive {
		return configSummary{
			source: "interactive",
			cfg:    cfg,
		}, nil
	}

	return configSummary{
		source: "none",
		cfg:    cfg,
	}, nil
}

func resolveConfig(ctx context.Context, streams ioStreams, opts installOptions, deps dependencies) (config.Config, error) {
	icons, iconErr := parseIconMode(opts.iconMode)
	if iconErr != nil {
		return config.Config{}, iconErr
	}

	if path, explicit := effectiveConfigPath(opts.configPath, opts.configExplicit); explicit {
		source := effectiveConfigSource(opts.configExplicit)
		verbosef(opts.verbose, streams.Err, "loading config from %s path %s", source, path)
		cfg, err := deps.loadConfig(path)
		if err != nil {
			return config.Config{}, explicitConfigError(path, err)
		}
		return cfg, nil
	}

	source, found, err := discoverConfig(opts.verbose, streams.Err, deps)
	if err != nil {
		return config.Config{}, err
	}
	if found {
		_, _ = fmt.Fprintf(streams.Err, "Using config %s\n", source.Path)
		return source.Config, nil
	}

	if !opts.interactive {
		return config.Config{}, noConfigError()
	}
	if !deps.isTerminal(streams) {
		return config.Config{}, fmt.Errorf("%w; --interactive requires stdin and stdout terminals", errNoConfig)
	}

	_, _ = fmt.Fprintln(streams.Err, "No config found. Starting interactive mode...")
	verbosef(opts.verbose, streams.Err, "loading releases for interactive picker")
	releases, err := tui.LoadReleases(ctx, deps.listReleases, streams.Err)
	if err != nil {
		return config.Config{}, err
	}

	defaults := defaultConfig()
	result, err := deps.runTUI(ctx, releases, tui.Options{
		Destination:      defaults.Destination,
		RefreshFontCache: defaults.RefreshFontCache,
		Icons:            icons,
	})
	if err != nil {
		return config.Config{}, err
	}
	if result.Cancelled {
		return config.Config{}, errCancelled
	}
	return result.Config, nil
}

func discoverConfig(verbose bool, stderr io.Writer, deps dependencies) (config.Source, bool, error) {
	if verbose {
		paths, err := deps.defaultConfigPaths()
		if err != nil {
			verbosef(true, stderr, "could not enumerate config candidates: %v", err)
		} else {
			for _, path := range paths {
				verbosef(true, stderr, "checking config candidate %s", path)
			}
		}
	}
	source, found, err := deps.discoverConfig()
	if err != nil {
		return config.Source{}, false, err
	}
	if found {
		verbosef(verbose, stderr, "discovered config %s", source.Path)
	} else {
		verbosef(verbose, stderr, "no config discovered")
	}
	return source, found, nil
}

func printLegacyFontNames(stdout io.Writer, release nerdfonts.Release) {
	_, _ = fmt.Fprintf(stdout, "# %s\nfamilies:\n", release.TagName)
	for _, family := range release.Families {
		_, _ = fmt.Fprintf(stdout, "  - %s\n", family)
	}
}

func printInfo(stdout io.Writer, payload infoOutput) {
	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Config path:\t%s\n", payload.ConfigPath)
	_, _ = fmt.Fprintf(w, "Config source:\t%s\n", payload.ConfigSource)
	_, _ = fmt.Fprintf(w, "Release selector:\t%s\n", payload.ReleaseSelector)
	_, _ = fmt.Fprintf(w, "Resolved release:\t%s\n", payload.ResolvedRelease)
	_, _ = fmt.Fprintf(w, "Destination:\t%s\n", payload.Destination)
	families := "(none)"
	if len(payload.Families) > 0 {
		families = strings.Join(payload.Families, ", ")
	}
	_, _ = fmt.Fprintf(w, "Families:\t%s\n", families)
	_, _ = fmt.Fprintf(w, "Refresh font cache:\t%t\n", payload.RefreshFontCache)
	_, _ = fmt.Fprintf(w, "Platform:\t%s/%s\n", payload.GoOS, payload.GoArch)
	_, _ = fmt.Fprintf(w, "Go version:\t%s\n", payload.GoVersion)
	_, _ = fmt.Fprintf(w, "Version:\t%s\n", payload.Version)
	_, _ = fmt.Fprintf(w, "Commit:\t%s\n", payload.Commit)
	_, _ = fmt.Fprintf(w, "Date:\t%s\n", payload.Date)
	_ = w.Flush()
}

func displayConfigPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "none found"
	}
	return path
}

func writeJSON(stdout io.Writer, payload any, stderr io.Writer) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		_, _ = fmt.Fprintf(stderr, "encode json: %v\n", err)
		return 1
	}
	return 0
}

func verbosef(enabled bool, stderr io.Writer, format string, args ...any) {
	if !enabled {
		return
	}
	_, _ = fmt.Fprintf(stderr, "verbose: "+format+"\n", args...)
}
