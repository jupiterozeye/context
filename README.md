# Context

Terminal context capture tool for AI-assisted debugging. Simplifies sharing terminal context with AI without manual copy-pasting.

## Features

- **`context dir [path]`** - Generate directory tree and copy to clipboard
- **`context last [n]`** - Copy last n terminal outputs to clipboard
- Cross-platform clipboard support (Linux, macOS, Windows)
- Shell integration for Bash, Zsh, and Fish
- Nix flake support

## Installation

### Using Nix

```bash
# Build and install
nix profile install github:jupiterozeye/context

# Or run without installing
nix run github:jupiterozeye/context -- dir ~/projects
```

### From Source

```bash
git clone https://github.com/jupiterozeye/context.git
cd context
go build -o context ./cmd/context
sudo cp context /usr/local/bin/
```

## Setup

### Shell Integration (Required for `context last`)

To capture terminal output history, add the appropriate line to your shell config:

**Bash** (`~/.bashrc`):
```bash
source /usr/local/share/context/shell/context.bash
```

**Zsh** (`~/.zshrc`):
```zsh
source /usr/local/share/context/shell/context.zsh
```

**Fish** (`~/.config/fish/config.fish`):
```fish
source /usr/local/share/context/shell/context.fish
```

Then restart your terminal or run:
```bash
source ~/.bashrc  # or ~/.zshrc
```

## Usage

### Directory Tree

```bash
# Current directory
context dir

# Specific directory
context dir ~/projects/myproject

# With options
context dir ~/projects --depth 2 --exclude "node_modules,.git"
context dir ~/projects -d 2 -e "node_modules,.git"
```

**Options:**
- `-d, --depth int` - Max depth (0 = unlimited)
- `-e, --exclude string` - Comma-separated patterns to exclude
- `-H, --hidden` - Include hidden files
- `-f, --format string` - Output format: tree|json|markdown (default: tree)
- `-c, --no-copy` - Print only, don't copy to clipboard

### Terminal History

```bash
# Copy last command output
context last

# Copy last 3 command outputs
context last 3

# With options
context last 5 --format markdown
```

**Options:**
- `-r, --raw` - Raw output without formatting
- `-f, --format string` - Output format: raw|command|markdown (default: raw)
- `-c, --no-copy` - Print only, don't copy to clipboard

## How It Works

### `context dir`

Walks the directory tree and generates a formatted tree structure, similar to the Unix `tree` command. The output is automatically copied to your clipboard.

### `context last`

The shell integration scripts hook into your shell's preexec/precmd hooks to capture:
- The command that was run
- The command's output (base64 encoded)
- Exit code
- Working directory
- Timestamp

This data is stored in `~/.local/share/context/history.jsonl` and read by the `context last` command.

## Development

```bash
# Run locally
go run ./cmd/context dir

# Build
go build -o context ./cmd/context

# Test
make test

# Build with Nix
nix build
```

## License

MIT