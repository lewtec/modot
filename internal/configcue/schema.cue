package modot

import "strings"

#Input: {
	from:    string
	version?: string
}

#Host: {
	ips?:  [...string]
	mac?:  string
	port?: int
	user?: string
}

#LazyTool: {
	version?: string
	ref?:     string
	pkg?:     string
	global?:  bool
	alias?:   string
	bins?:    [...string]
}

// core:place step: exact-path move on the origin virtual FS (file or dir prefix).
#PlaceStepMove: {
	op:   "move"
	from: string
	to:   string
}

// core:place step: gitignore-style require clauses (AND). "!" = must not match.
#PlaceStepRequire: {
	op: "require"
	patterns: [string]: string
}

#PlaceStep: #PlaceStepMove | #PlaceStepRequire

// Legacy core:place module config. The prelude copies this onto file.home.
#PlaceItems: [string]: string
#PlaceSteps: [string]: #PlaceStep

#PlaceConfig: {
	items: #PlaceItems | *{}
	ignore_missing?: bool
	steps: #PlaceSteps | *{}
	...
}

#ModuleRef: {
	enable: bool | *true
	input?: string
	path?:  string | *"."
	// Default "" so guards can reference from without "optional field" errors
	// (many modules only set input/path and omit from).
	from:     string | *""
	version?: string
	config: _ | *{}
	// Legacy from: "core:place" is checked here, then rewritten below.
	if from == "core:place" {
		config: #PlaceConfig
	}
}

// from: "core:place" becomes file.home place entries. The module is turned off.
file: home: {
	for name, mod in modules if mod.from == "core:place" for dest, src in mod.config.items {
		let path = strings.Trim(dest, "/")
		if path != "" {
			"\(path)": {
				type: "place"
				source: "\(name)": src
				steps: mod.config.steps
			}
		}
	}
}

modules: {
	for name, mod in modules if mod.from == "core:place" {
		"\(name)": enable: false
	}
}

#Runtime: {
	is_phone?: bool
	hostname?: string
	home?: string
	dotfiles_root?: string
	config_dir?: string
	user_data_dir?: string
	cpus?: int
	goos?: string
	goarch?: string
	memory?: int
	// Command family that loaded this config.
	mode?: "home" | "codebase" | "system"
	inputs?: [string]: {
		path?: string
		provider?: string
		target?: string
	}
}

#Concurrency: {
	io?:       int
	cpu?:      int
	internet?: int
}

#BackupActionGitRepoSync: close({
	name?: string
	kind: "git_repo_sync"
	src:  string
	dst:  string
})

#BackupActionRsync: close({
	name?: string
	kind: "rsync"
	src:  string
	dst:  string
	excludes?: [...string]
	skip_permissions?: bool
})

#BackupActionArchive: close({
	name?: string
	kind:      "archive"
	input_dir: string
	output:    string
	format:    "tar"
})


#BackupAction: #BackupActionGitRepoSync | #BackupActionRsync | #BackupActionArchive

inputs: {
	self: #Input & {
		from: "self"
	}
	[string]: #Input
}
runtime?: #Runtime
// dest profiles: file.<profile> is a compose #Tree
modules: {
	[string]: #ModuleRef
}
workspaces?: [string]: int
desktop?: {
	dark_mode?: bool
	raw?: {
		dconf?: [string]: [string]: _
	}
	wallpaper?: {
		dir?:     string
		default?: string
	}
}
screenshot?: {
	dir?: string
}
hosts?: [string]: #Host
backup?: {
	rsyncnet_user?: string
	remote_path?:   string
	git_repos?: [...{
		src: string
		dst: string
	}]
	actions?: [...#BackupAction]
}
quicksync?: {
	repo_dir?:    string
	remote_path?: string
}
browser?: {
	default?: string
	webapp?:  string
}
lazy_tools?: [string]: #LazyTool
drivers?: [string]: [string]: int
concurrency?: #Concurrency

// LSP router: language servers behind `modot codebase lsp`.
// Empty / omitted means the proxy still speaks LSP but routes nowhere.
lsp?: #LSP

// Linters for `modot codebase lint` (CUE-defined tools + codecs).
lint?: #Checks
// Formatters for `modot codebase format`.
formatter?: #Checks

// #Checks is a map of named check tools (lint or formatter).
#Checks: {
	tools?: [string]: #CheckTool
}

// #CheckTool is one linter or formatter declaration.
#CheckTool: {
	// Off-switch without deleting the entry (default on).
	enable: bool | *true
	// Ordered firewall rules (keys like "00-go-mod"); first match wins.
	detect?: [string]: #DetectRule
	// lazy_tools names to ensure before run (map for deep-merge).
	needs?: [string]: bool
	// argv (format tools should include write flags).
	cmd: [...string] & [_, ...]
	// Lint only: codec name (sarif, actionlint_json, shellcheck_json, eslint_json).
	output?: string
	// When true, append files matching the winning detect rule's glob to cmd.
	args_from_globs?: bool
}

// #DetectRule is one firewall entry for tool applicability.
#DetectRule: {
	// Match if this path exists under the run root (file or directory).
	path?: string
	// Match if any file under the root matches this glob (** and {a,b} supported).
	glob?: string
	// Outcome when this rule is the first match.
	enable: bool
}

// #LSP is the codebase-local language server router config.
#LSP: {
	// extension (with or without leading dot) -> our language id
	extensions?: [string]: string
	// editor languageId -> our language id (used when extension map misses)
	language_ids?: [string]: string
	// our language -> ordered server attachments (keys like "00_gopls")
	languages?: [string]: #LSPLanguage
	// server id -> process definition
	servers?: [string]: #LSPServer
	// soft timeout per backend request (Go duration string), default applied in code
	request_timeout?: string
}

#LSPLanguage: {
	// ordered attachment key -> capabilities this server may handle for the language
	[string]: #LSPAttachment
}

#LSPAttachment: {
	// capability flags (hover, definition, diagnostics, …). Empty = all.
	capabilities?: [string]: bool
}

#LSPServer: {
	// argv for the language server (stdio)
	cmd: [...string] & [_, ...]
	// lazy_tools names to ensure before spawn (map for deep-merge)
	needs?: [string]: bool
}
