package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

type helpTopic struct {
	command   string
	printHelp func(io.Writer)
}

func (h helpTopic) usageHint() string {
	command := commandName
	if h.command != "" {
		command += " " + h.command
	}
	return fmt.Sprintf("Run '%s --help' for usage.", command)
}

func handleParseError(err error, streams ioStreams, help helpTopic) int {
	if errors.Is(err, flag.ErrHelp) {
		help.printHelp(streams.Out)
		return 0
	}
	return usageError(streams, help, err.Error())
}

func usageError(streams ioStreams, help helpTopic, message string) int {
	_, _ = fmt.Fprintf(streams.Err, "%s\n%s\n", message, help.usageHint())
	return 2
}

func printRootHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `%s installs Nerd Fonts from a config file or an interactive picker.

Usage:
  %s [flags]
  %s install [flags]
  %s list [fonts] [flags]
  %s info [flags]
  %s version [flags]
  %s completion [bash|zsh]

Commands:
  install     Install fonts from config (default when no subcommand is given)
  list        List available Nerd Font family names for a release
  info        Print resolved configuration and build diagnostics
  version     Print version, build, and runtime information
  completion  Print a shell completion script

Global flags:
  --config <path>   Use a specific config file
  --verbose, -v     Print extra diagnostics to stderr
  --help, -h        Show this help text

Install flags:
  --dry-run         Print the plan without installing fonts
  --interactive     Start the picker when no config file is found
  --icons <mode>    Picker icons: auto, nerd, unicode, ascii

Output flags:
  --json            Print JSON for list, info, and version
  --release <tag>   Select a release for list or --font-names

Compatibility flags:
  --font-names      Deprecated alias for 'list' (YAML by default)
  --version         Alias for 'version'

Examples:
  %s --dry-run
  %s list --release v3.4.0 --json
  %s info
`, commandName, commandName, commandName, commandName, commandName, commandName, commandName, commandName, commandName, commandName)
}

func printInstallHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `Install Nerd Fonts from config or from the interactive picker.

Usage:
  %s install [flags]
  %s [flags]

Flags:
  --config <path>   Use a specific config file
  --dry-run         Print the plan without installing fonts
  --interactive     Start the picker when no config file is found
  --icons <mode>    Picker icons: auto, nerd, unicode, ascii
  --verbose, -v     Print extra diagnostics to stderr
  --help, -h        Show this help text

Examples:
  %s install --config fonts.yaml --dry-run
  %s --interactive --icons ascii
`, commandName, commandName, commandName, commandName)
}

func printListHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `List available Nerd Font family names for a resolved release.

Usage:
  %s list [fonts] [flags]

Flags:
  --config <path>   Use a specific config file to resolve the default release
  --release <tag>   Release tag to inspect (defaults to config or latest)
  --json            Print JSON with the resolved release and families
  --verbose, -v     Print extra diagnostics to stderr
  --help, -h        Show this help text

Examples:
  %s list
  %s list fonts --json
  %s --font-names
`, commandName, commandName, commandName, commandName)
}

func printInfoHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `Print resolved configuration and build diagnostics.

Usage:
  %s info [flags]

Flags:
  --config <path>   Use a specific config file
  --interactive     Report interactive defaults when no config is found
  --json            Print machine-readable JSON
  --verbose, -v     Print extra diagnostics to stderr
  --help, -h        Show this help text

Examples:
  %s info
  %s info --config fonts.yaml --json
`, commandName, commandName, commandName)
}

func printVersionHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `Print version, build, and runtime information.

Usage:
  %s version [flags]

Flags:
  --json       Print machine-readable JSON
  --help, -h   Show this help text

Examples:
  %s version
  %s version --json
`, commandName, commandName, commandName)
}

func printCompletionHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `Print a shell completion script.

Usage:
  %s completion [bash|zsh]

Examples:
  %s completion bash
  %s completion zsh
`, commandName, commandName, commandName)
}

func completionScript(shell string) (string, error) {
	funcName := strings.ReplaceAll(commandName, "-", "_")
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		return fmt.Sprintf(`#!/usr/bin/env bash

_%[1]s_completion() {
  local cur prev words cword
  _init_completion || return

  local commands="install list info version completion"
  local global_flags="--config --verbose -v --help -h"

  case "${words[1]}" in
    install)
      COMPREPLY=( $(compgen -W "--config --dry-run --interactive --icons --verbose -v --help -h" -- "$cur") )
      return
      ;;
    list)
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "fonts --config --release --json --verbose -v --help -h" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "--config --release --json --verbose -v --help -h" -- "$cur") )
      fi
      return
      ;;
    info)
      COMPREPLY=( $(compgen -W "--config --interactive --json --verbose -v --help -h" -- "$cur") )
      return
      ;;
    version)
      COMPREPLY=( $(compgen -W "--json --help -h" -- "$cur") )
      return
      ;;
    completion)
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
      fi
      return
      ;;
  esac

  if [[ $cword -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "$commands $global_flags --dry-run --interactive --icons --font-names --version --release --json" -- "$cur") )
    return
  fi

  COMPREPLY=( $(compgen -W "$global_flags" -- "$cur") )
}

complete -F _%[1]s_completion %[2]s
`, funcName, commandName), nil
	case "zsh":
		return fmt.Sprintf(`#compdef %s

local -a commands
commands=(
  'install:install fonts from config'
  'list:list available family names'
  'info:print resolved configuration'
  'version:print version information'
  'completion:print completion scripts'
)

_arguments -C \
  '--config[use a specific config file]:config file:_files' \
  '--verbose[print extra diagnostics to stderr]' \
  '-v[print extra diagnostics to stderr]' \
  '--help[show help]' \
  '-h[show help]' \
  '1:command:->command' \
  '*::arg:->args'

case $state in
  command)
    _describe 'command' commands
    ;;
  args)
    case $words[2] in
      install)
        _arguments \
          '--config[use a specific config file]:config file:_files' \
          '--dry-run[print the plan without installing fonts]' \
          '--interactive[start the picker when no config is found]' \
          '--icons[picker icon mode]:mode:(auto nerd unicode ascii)' \
          '--verbose[print extra diagnostics to stderr]' \
          '-v[print extra diagnostics to stderr]' \
          '--help[show help]' \
          '-h[show help]'
        ;;
      list)
        _arguments \
          '1:target:(fonts)' \
          '--config[use a specific config file]:config file:_files' \
          '--release[release tag to inspect]:release:' \
          '--json[print JSON]' \
          '--verbose[print extra diagnostics to stderr]' \
          '-v[print extra diagnostics to stderr]' \
          '--help[show help]' \
          '-h[show help]'
        ;;
      info)
        _arguments \
          '--config[use a specific config file]:config file:_files' \
          '--interactive[report interactive defaults]' \
          '--json[print JSON]' \
          '--verbose[print extra diagnostics to stderr]' \
          '-v[print extra diagnostics to stderr]' \
          '--help[show help]' \
          '-h[show help]'
        ;;
      version)
        _arguments '--json[print JSON]' '--help[show help]' '-h[show help]'
        ;;
      completion)
        _arguments '1:shell:(bash zsh)'
        ;;
    esac
    ;;
esac
`, commandName), nil
	default:
		return "", fmt.Errorf("unsupported shell %q; use bash or zsh", shell)
	}
}
