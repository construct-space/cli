# Merge Plan: construct-cli + construct-tui → `construct`

## Why

Two separate Go binaries that share the same operator ecosystem, same auth, same project
model. Users install both. The CLI already distributes via npm; adding TUI capabilities
to the same binary means one install, one update path, one `construct` command.

---

## Current State

| | construct-cli | construct-tui |
|---|---|---|
| Module | `github.com/construct-space/cli` | `construct-tui` |
| Go | 1.24 | 1.25 |
| Framework | cobra + bubbletea/lipgloss | tview/tcell |
| Commands | 13 (scaffold, build, dev, publish...) | 25+ (interactive shell: /mode, /vibe, /architect...) |
| Distribution | npm (`@construct-space/cli`) + Go binary | standalone binary |
| Operator | none (space tooling only) | TCP client, streaming, session state |
| Auth | OAuth login, credential storage | imports from codex auth |
| Internal pkgs | manifest, entry, runtime*, auth, agent, copy, publish, ui, watcher | runtime*, shell |

\* Both have `internal/runtime/` but they're **completely different**:
- CLI: JS runtime detection (bun/deno/node)
- TUI: operator TCP client + stream normalization + session state

---

## Merged CLI Surface

```
construct                           # (no args) show help
construct version                   # version info

# ── Space Development (existing CLI) ────────────────────────
construct space new                 # scaffold new space (alias: construct new)
construct space build               # generate entry + vite build
construct space dev                 # dev mode with file watching
construct space run                 # install space locally
construct space validate            # validate manifest
construct space check               # typecheck + lint
construct space publish             # publish to registry
construct space clean               # remove build artifacts

# Top-level aliases preserved for muscle memory:
construct build                     # → construct space build
construct dev                       # → construct space dev
construct new                       # → construct space new

# ── Operator Interaction (from TUI) ────────────────────────
construct tui                       # launch full interactive TUI
construct chat "message"            # one-shot chat (stream to stdout)
construct vibe "goal"               # start vibe session (stream to stdout)
construct agent "task"              # dispatch agent task

# ── Auth ────────────────────────────────────────────────────
construct login                     # browser OAuth
construct logout                    # clear credentials

# ── Utilities ───────────────────────────────────────────────
construct update                    # self-update
construct models                    # list available models
construct agents                    # list available agents
construct providers                 # list configured providers
```

---

## Directory Structure

```
construct-cli/                      # repo stays, module name stays
├── main.go                         # embed templates, cmd.Execute()
├── go.mod                          # github.com/construct-space/cli, Go 1.25
├── go.sum
├── release.sh                      # updated for merged binary
│
├── cmd/                            # cobra commands
│   ├── root.go                     # root + top-level aliases (build, dev, new)
│   ├── version.go
│   ├── update.go
│   │
│   ├── space.go                    # `construct space` parent command
│   ├── space_build.go              # construct space build
│   ├── space_dev.go                # construct space dev
│   ├── space_scaffold.go           # construct space new / scaffold
│   ├── space_run.go
│   ├── space_validate.go
│   ├── space_check.go
│   ├── space_publish.go
│   ├── space_clean.go
│   │
│   ├── tui.go                      # construct tui (interactive TUI)
│   ├── chat.go                     # construct chat "msg"
│   ├── vibe.go                     # construct vibe "goal"
│   ├── agent.go                    # construct agent "task"
│   ├── models.go                   # construct models
│   ├── agents.go                   # construct agents
│   ├── providers.go                # construct providers
│   │
│   ├── login.go                    # construct login
│   └── logout.go                   # construct logout
│
├── internal/
│   ├── manifest/                   # space manifest read/write/validate (from CLI)
│   │   └── manifest.go
│   │
│   ├── entry/                      # entry.ts generation (from CLI)
│   │   └── entry.go
│   │
│   ├── jsruntime/                  # ← RENAMED from CLI's internal/runtime/
│   │   └── runtime.go             #   JS runtime detection (bun/deno/node)
│   │
│   ├── operator/                   # ← FROM TUI's internal/runtime/
│   │   ├── client.go              #   TCP client to construct-operator
│   │   ├── stream.go              #   stream normalization, payload builders
│   │   └── types.go               #   SessionState, Events, Modes, Transcript
│   │
│   ├── shell/                      # ← FROM TUI's internal/shell/
│   │   ├── commands.go            #   /slash command parsing
│   │   ├── state.go               #   state mutation helpers
│   │   ├── architect.go           #   architect interview system
│   │   └── examples.go            #   built-in vibe examples
│   │
│   ├── auth/                       # OAuth + credential storage (from CLI)
│   │   └── auth.go
│   │
│   ├── agent/                      # agent file XOR encoding (from CLI)
│   │   └── agent.go
│   │
│   ├── publish/                    # source packing for registry (from CLI)
│   │   └── publish.go
│   │
│   ├── copy/                       # recursive dir copy (from CLI)
│   │   └── copy.go
│   │
│   ├── watcher/                    # fsnotify with debounce (from CLI)
│   │   └── watcher.go
│   │
│   └── ui/                         # terminal UI (merge both)
│       ├── styles.go              #   lipgloss styles, Success/Error/Warn (from CLI)
│       ├── spinner.go             #   bubbletea spinner (from CLI)
│       ├── dev.go                 #   space dev TUI (from CLI)
│       └── tui.go                 #   operator interactive TUI (from TUI main.go)
│
├── src/                            # TypeScript (unchanged)
│   └── build/
│       ├── vite-preset.ts
│       ├── host-api.ts
│       └── externals.ts
│
├── npm/                            # npm distribution (unchanged)
│   ├── package.json
│   ├── install.cjs
│   ├── bin/construct
│   ├── binaries/
│   └── dist/
│
├── templates/                      # space templates (unchanged)
│   └── space/
│
└── examples/                       # ← FROM TUI: built-in vibe examples
    ├── flutter.md
    ├── go.md
    ├── rails.md
    ├── nuxt.md
    └── next.md
```

---

## Migration Steps

### Phase 1: Prepare construct-cli (non-breaking)

1. **Rename `internal/runtime/` → `internal/jsruntime/`**
   - Update all imports in cmd/*.go
   - No external API change, purely internal

2. **Namespace space commands under `construct space`**
   - Create `cmd/space.go` as parent command
   - Move build/dev/scaffold/validate/check/publish/run/clean under it
   - Keep top-level aliases in root.go for backwards compat:
     ```go
     rootCmd.AddCommand(spaceCmd)
     // aliases
     rootCmd.AddCommand(aliasCmd("build", spaceBuildCmd))
     rootCmd.AddCommand(aliasCmd("dev", spaceDevCmd))
     rootCmd.AddCommand(aliasCmd("new", spaceScaffoldCmd))
     ```

3. **Bump Go version to 1.25**

4. **Test everything still works.** npm install, build, dev, publish — all unchanged.

### Phase 2: Move TUI code in

5. **Copy `construct-tui/internal/runtime/` → `internal/operator/`**
   - Rename package declaration: `package operator`
   - Update type references (no external consumers, clean move)

6. **Copy `construct-tui/internal/shell/` → `internal/shell/`**
   - Update imports to use `internal/operator` instead of `internal/runtime`

7. **Copy `construct-tui/examples/` → `examples/`**

8. **Extract TUI app from `construct-tui/main.go` → `internal/ui/tui.go`**
   - Refactor `tuiApp` to be launchable via `func RunTUI(addr, clientID string) error`
   - Update imports to `internal/operator`, `internal/shell`

9. **Add new tview/tcell dependencies to go.mod**

### Phase 3: Wire up cobra commands

10. **`cmd/tui.go`**: Launch interactive TUI
    ```go
    // construct tui [--addr host:port]
    func runTUI(cmd *cobra.Command, args []string) error {
        return ui.RunTUI(addr, clientID)
    }
    ```

11. **`cmd/chat.go`**: One-shot chat (no TUI needed)
    ```go
    // construct chat "what is construct?"
    // Connects to operator, streams response to stdout, exits
    ```

12. **`cmd/vibe.go`**: Start vibe session
    ```go
    // construct vibe "build a todo app" [--project ./my-app] [--model claude-sonnet-4-6]
    // Connects to operator, streams events to stdout, exits on done
    ```

13. **`cmd/agent.go`**: Dispatch agent task
    ```go
    // construct agent "analyze this codebase" [--agent-id general]
    ```

14. **`cmd/models.go`**, **`cmd/agents.go`**, **`cmd/providers.go`**: List resources from operator

### Phase 4: Shared auth

15. **Wire operator commands to use `internal/auth`**
    - TUI currently imports from codex auth; switch to construct's own credential store
    - `construct login` → credentials available to both space and operator commands
    - Provider key import (`/provider import-auth`) uses same `internal/auth` path

### Phase 5: Cleanup

16. **Update `release.sh`** — single binary, same npm distribution
17. **Update npm `package.json`** — same binary now has operator capabilities
18. **Delete `construct-tui/` repo** (archive, don't hard delete)
19. **Move tests** — all test files follow their source files

---

## Package Dependency Graph (after merge)

```
cmd/
 ├── space_*.go     → manifest, entry, jsruntime, auth, agent, publish, watcher, ui
 ├── tui.go         → ui (tui), operator, shell
 ├── chat.go        → operator
 ├── vibe.go        → operator
 ├── agent.go       → operator
 ├── login.go       → auth
 └── models.go      → operator

internal/
 ├── operator/      → (net, encoding/json, sync — no internal deps)
 ├── shell/         → operator
 ├── jsruntime/     → (os/exec — no internal deps)
 ├── manifest/      → (os, encoding/json — no internal deps)
 ├── entry/         → manifest
 ├── auth/          → (os, encoding/json — no internal deps)
 ├── agent/         → copy
 ├── publish/       → (os, archive/tar — no internal deps)
 ├── watcher/       → (fsnotify — no internal deps)
 ├── copy/          → (os — no internal deps)
 └── ui/
      ├── styles.go  → (lipgloss)
      ├── spinner.go → (bubbletea)
      ├── dev.go     → (bubbletea, lipgloss)
      └── tui.go     → (tview, tcell), operator, shell
```

No circular dependencies. `operator/` and `shell/` remain fully reusable by
the Vibe space (as planned in VIBE_RUNTIME_PROTOTYPE_PLAN.md).

---

## What Doesn't Change

- **npm package name**: `@construct-space/cli`
- **Module path**: `github.com/construct-space/cli`
- **Vite preset**: `@construct-space/cli/dist/build/vite-preset.js`
- **Space manifest format**: `space.manifest.json`
- **Operator protocol**: TCP newline-delimited JSON
- **Existing commands**: `construct build`, `construct dev`, etc. still work
- **Templates**: unchanged
- **Release flow**: same cross-platform build + npm publish

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Binary size increases (~tview/tcell deps) | Acceptable; single binary is worth the trade |
| Two TUI frameworks (bubbletea + tview) | They coexist fine; bubbletea for inline spinners, tview for full-screen TUI |
| Breaking existing `construct` installs | Phase 1 is non-breaking; aliases preserve all existing commands |
| Go 1.25 not available everywhere | Already required by construct-tui; CI uses latest |

---

## Execution Order

```
Phase 1  ██████░░░░░░░░░░  Prepare CLI (rename, namespace, bump Go)
Phase 2  ░░░░░░██████░░░░  Move TUI code in (operator, shell, ui, examples)
Phase 3  ░░░░░░░░░░████░░  Wire cobra commands (tui, chat, vibe, agent)
Phase 4  ░░░░░░░░░░░░██░░  Shared auth
Phase 5  ░░░░░░░░░░░░░░██  Cleanup (release, npm, archive old repo)
```

Each phase is independently testable and committable.
