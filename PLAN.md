Architecture and Implementation Plan for `context`
==================================================

## 1. Terminal Output Capture Solution (COMPLETED)

### Selected Approach: Shell Integration with Log File

**Mechanism:**
- Users source a shell script (`context.bash`, `context.zsh`, `context.fish`) in their shell config
- Script uses `preexec`/`precmd` hooks to capture command output
- Output is written to structured log file: `~/.local/share/context/history.jsonl`
- Go CLI reads from this log file for `context last` command

**Log File Format (JSON Lines):**
```json
{"timestamp": "2026-02-05T11:03:16Z", "command": "ls -la", "output": "...", "exit_code": 0, "pwd": "/home/user"}
```

**Pros:**
- Works across bash/zsh/fish
- No PTY wrapping needed
- Captures actual terminal output
- Persistent history

**Cons:**
- Requires shell setup (one-time)
- Users must source the script
- Log file grows (solved with rotation)

---

## 2. CLI Architecture

### Command Structure

```
context [command] [flags] [args]

Commands:
  dir [path]     Generate directory tree and copy to clipboard
  last [n]       Copy last n terminal outputs to clipboard
  version        Print version information
  init           Print shell integration setup instructions

Global Flags:
  -h, --help     Show help
  -v, --verbose  Verbose output
```

### Command: `dir`

```bash
context dir [path] [flags]

Arguments:
  path           Directory path (default: current directory)

Flags:
  -d, --depth int       Max depth (default: unlimited)
  -e, --exclude string  Comma-separated patterns to exclude (e.g., "node_modules,.git")
  -H, --hidden          Include hidden files (default: false)
  -f, --format string   Output format: tree|json|markdown (default: tree)
  -c, --no-copy         Print only, don't copy to clipboard
```

**Output Example:**
```
myproject/
├── cmd/
│   └── context/
│       └── main.go
├── internal/
│   ├── dir/
│   │   ├── tree.go
│   │   └── clipboard.go
│   └── last/
│       ├── reader.go
│       └── parser.go
├── go.mod
├── go.sum
└── README.md

Copied to clipboard!
```

### Command: `last`

```bash
context last [n] [flags]

Arguments:
  n              Number of previous outputs (default: 1)

Flags:
  -r, --raw             Raw output without formatting
  -f, --format string   Output format: raw|command|markdown (default: raw)
  -c, --no-copy         Print only, don't copy to clipboard
  --since duration      Get outputs since time (e.g., "5m", "1h")
```

**Output Example:**
```
=== Command 1: ls -la ===
total 64
drwxr-xr-x  6 user user 4096 Feb 5 10:00 .
drwxr-xr-x 20 user user 4096 Feb 5 09:00 ..

=== Command 2: go test ./... ===
ok      github.com/user/context/cmd/context 0.234s
ok      github.com/user/context/internal/dir 0.123s

Copied to clipboard!
```

---

## 3. Repository Structure

```
context/
├── cmd/
│   └── context/
│       └── main.go              # CLI entry point
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root command setup (cobra)
│   │   ├── dir.go               # Dir command
│   │   ├── last.go              # Last command
│   │   └── version.go           # Version command
│   ├── dir/
│   │   ├── tree.go              # Tree generation logic
│   │   ├── filter.go            # File filtering
│   │   └── formatter.go         # Output formatters (tree, json, markdown)
│   ├── last/
│   │   ├── reader.go            # Read history from log file
│   │   ├── parser.go            # Parse JSONL entries
│   │   └── formatter.go         # Format output
│   ├── clipboard/
│   │   └── clipboard.go         # Cross-platform clipboard operations
│   └── shell/
│       └── installer.go         # Shell integration setup helpers
├── shell/
│   ├── context.bash             # Bash integration script
│   ├── context.zsh              # Zsh integration script
│   └── context.fish             # Fish integration script
├── pkg/
│   └── config/
│       └── config.go            # Configuration management
├── scripts/
│   └── install.sh               # Installation script
├── flake.nix                    # Nix flake
├── flake.lock                   # Nix flake lock
├── default.nix                  # Nix package definition
├── go.mod                       # Go module
├── go.sum                       # Go dependencies
├── README.md                    # Documentation
├── LICENSE                      # License
├── .gitignore                   # Git ignore
└── Makefile                     # Build automation
```

### Go Module Structure

```go
// go.mod
module github.com/yourusername/context

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/atotto/clipboard v0.1.4
    github.com/fatih/color v1.16.0
)
```

---

## 4. Nix Flake Design

### flake.nix

```nix
{
  description = "Context - Terminal context capture tool for AI-assisted debugging";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          context = pkgs.buildGoModule {
            pname = "context";
            version = "0.1.0";
            src = ./.;
            vendorHash = null;  # Will be set after go mod vendor
            
            meta = with pkgs.lib; {
              description = "Terminal context capture tool for AI-assisted debugging";
              homepage = "https://github.com/yourusername/context";
              license = licenses.mit;
              maintainers = [ ];
            };
          };
          default = self.packages.${system}.context;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gofumpt
            golangci-lint
          ];
        };
      });
}
```

### Build Commands

```bash
# Build locally
nix build

# Build from GitHub
nix build github:yourusername/context

# Run directly
nix run github:yourusername/context -- dir /some/path
nix run github:yourusername/context -- last -3

# Install
nix profile install github:yourusername/context
```

---

## 5. Implementation Milestones

### Phase 1: Foundation (Day 1)

**Tasks:**
1. Initialize Go module
2. Set up cobra CLI framework
3. Implement `dir` command with tree generation
4. Implement clipboard integration
5. Add basic filtering and formatting

**Deliverables:**
- `context dir` works with tree output
- Copies to clipboard
- Handles current directory default

### Phase 2: Shell Integration (Days 2-3)

**Tasks:**
1. Create shell hook scripts (bash, zsh, fish)
2. Implement JSONL log file writer in shell
3. Implement `last` command reader
4. Add log rotation (keep last 1000 commands)
5. Create `context init` command for setup instructions

**Deliverables:**
- Shell scripts capture command output
- `context last` reads from log file
- Users can source shell integration

### Phase 3: Nix & Polish (Day 4)

**Tasks:**
1. Create flake.nix
2. Create default.nix for legacy nix-build
3. Add comprehensive README
4. Add Makefile with common tasks
5. Test cross-platform clipboard support
6. Add --help documentation

**Deliverables:**
- `nix build github:owner/repo` works
- Full documentation
- Ready for use

### Phase 4: Advanced Features (Future)

- Config file support (~/.config/context/config.yaml)
- Custom exclude patterns
- Output to file option
- Integration with popular terminals
- Plugin system for custom formatters

---

## 6. Technical Details

### Clipboard Support

**Cross-platform:**
- Linux: `xclip` or `wl-clipboard` (fallback)
- macOS: `pbcopy` builtin
- Windows: `clip` builtin

**Implementation:** Use `github.com/atotto/clipboard`

### Tree Generation

**Algorithm:**
1. Walk directory recursively
2. Sort entries (dirs first, then files)
3. Build tree structure with prefixes (`├──`, `└──`, `│   `)
4. Apply filters (hidden files, exclude patterns)
5. Respect depth limit

**Example Output:**
```
myproject/
├── cmd/
│   └── context/
│       └── main.go
├── go.mod
└── README.md
```

### Shell Integration Scripts

**Bash:**
```bash
# ~/.bashrc
source /path/to/context/shell/context.bash
```

**Zsh:**
```zsh
# ~/.zshrc
source /path/to/context/shell/context.zsh
```

**Fish:**
```fish
# ~/.config/fish/config.fish
source /path/to/context/shell/context.fish
```

---

## 7. Usage Examples

### Basic Usage

```bash
# Get current directory tree
context dir

# Get specific directory tree
context dir ~/projects/myproject

# Get last command output
context last

# Get last 3 commands
context last -3

# Custom depth and exclude
context dir ~/projects --depth 2 --exclude "node_modules,.git"
```

### With Nix

```bash
# One-off usage
nix run github:yourusername/context -- dir ~/projects

# Install permanently
nix profile install github:yourusername/context
context dir
context last -5
```

---

## Summary

This design provides:
1. ✅ Practical solution for `context last` via shell integration
2. ✅ Clean, extensible Go CLI architecture
3. ✅ Full Nix support with flakes
4. ✅ Cross-platform clipboard support
5. ✅ Comprehensive shell support (bash, zsh, fish)
6. ✅ Clear implementation phases

Ready to implement!
