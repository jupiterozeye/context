# Context

A terminal context capture tool that simplifies sharing your terminal context with AI assistants.

No more copy-pasting! Context automatically captures directory structures and command outputs, then copies them to your clipboard for easy sharing with AI tools.

## Features

- **`context dir [path]`** - Generate directory tree and copy to clipboard
- **`context last [n]`** - Copy last n terminal outputs to clipboard
- **`context init`** - Show shell integration setup instructions
- **`context version`** - Show version information
- Cross-platform clipboard support (Linux, macOS, Windows)
- Multiple output formats: tree, JSON, markdown
- Shell integration for Bash, Zsh, and Fish
- Nix flake support for easy installation

## Why Use Context?

Sharing terminal context with AI assistants usually means:
1. Running `ls -R` or `tree` and copy-pasting
2. Scrolling up to find command output and copy-pasting
3. Repeating this process multiple times

**Context automates this:** Just run `context dir` or `context last 3` and everything is copied to your clipboard, ready to paste into ChatGPT, Claude, or any AI assistant.

## Quick Start

```bash
# Try it without installing (requires Nix with flakes)
nix run github:jupiterozeye/context -- dir ~/projects

# Or install it
nix profile install github:jupiterozeye/context

# Set up shell integration (required for 'context last')
context init  # Shows setup instructions
```

**Common workflows:**
```bash
# Share your project structure with AI
context dir ~/my-project

# Share the last error message you got
context last

# Share multiple command outputs for debugging
context last 5 --format markdown
```

## Installation

### Method 1: Using Nix (Recommended)

**One-off usage (no installation):**
```bash
nix run github:jupiterozeye/context -- dir ~/projects
nix run github:jupiterozeye/context -- last 3
```

**Install to your profile:**
```bash
nix profile install github:jupiterozeye/context
```

### Method 2: From Source (requires Go)

```bash
# Clone and build
git clone https://github.com/jupiterozeye/context.git
cd context
go build -o context ./cmd/context

# Install to ~/.local/bin (recommended)
mkdir -p ~/.local/bin ~/.local/share/context/shell
cp context ~/.local/bin/
cp -r shell/* ~/.local/share/context/shell/

# Make sure ~/.local/bin is in your PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Method 3: Using Make (from source)

```bash
git clone https://github.com/jupiterozeye/context.git
cd context

# Install to ~/.local (recommended for NixOS users)
make build
mkdir -p ~/.local/bin ~/.local/share/context/shell
cp context ~/.local/bin/
cp -r shell/* ~/.local/share/context/shell/

# Or install system-wide (requires sudo, not recommended for NixOS)
sudo make install
```

## Advanced Installation

### NixOS Configuration (flakes)

Add to your system's `flake.nix`:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    context.url = "github:jupiterozeye/context";
  };

  outputs = { self, nixpkgs, context, ... }@inputs: {
    nixosConfigurations.yourhostname = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = { inherit inputs; };
      modules = [
        ./configuration.nix
        ({ pkgs, inputs, ... }: {
          environment.systemPackages = [ inputs.context.packages.x86_64-linux.default ];
          
          # Optional: Auto-source shell integration for all users
          programs.bash.interactiveShellInit = ''
            source ${inputs.context.packages.x86_64-linux.default}/share/context/shell/context.bash
          '';
          programs.zsh.interactiveShellInit = ''
            source ${inputs.context.packages.x86_64-linux.default}/share/context/shell/context.zsh
          '';
        })
      ];
    };
  };
}
```

Or in your `configuration.nix` (if using specialArgs):

```nix
{ config, pkgs, inputs, ... }:

{
  environment.systemPackages = [ inputs.context.packages.x86_64-linux.default ];
}
```

### Home Manager (flakes)

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    home-manager.url = "github:nix-community/home-manager";
    context.url = "github:jupiterozeye/context";
  };

  outputs = { nixpkgs, home-manager, context, ... }: {
    homeConfigurations.username = home-manager.lib.homeManagerConfiguration {
      pkgs = nixpkgs.legacyPackages.x86_64-linux;
      modules = [
        {
          home.packages = [ context.packages.x86_64-linux.context ];
          
          # Shell integration
          programs.bash.initExtra = ''
            source ${context.packages.x86_64-linux.context}/share/context/shell/context.bash
          '';
          
          programs.zsh.initExtra = ''
            source ${context.packages.x86_64-linux.context}/share/context/shell/context.zsh
          '';
        }
      ];
    };
  };
}
```



## Setup

### Shell Integration (Required for `context last`)

The shell integration captures command output so `context last` can access it.

**Quick setup:** Run `context init` to see setup instructions for your installation method.

**Manual setup:**

Choose the appropriate path based on how you installed:

| Installation Method | Shell Config Location | Command to Add |
|---------------------|----------------------|----------------|
| Nix profile install | `~/.bashrc` / `~/.zshrc` | `source ~/.nix-profile/share/context/shell/context.bash` |
| From source / Make | `~/.bashrc` / `~/.zshrc` | `source ~/.local/share/context/shell/context.bash` |
| System-wide (`sudo make install`) | `~/.bashrc` / `~/.zshrc` | `source /usr/local/share/context/shell/context.bash` |

Replace `.bash` with `.zsh` for Zsh or `.fish` for Fish shell, and update the config file location accordingly.

**Apply changes:**
```bash
source ~/.bashrc  # or ~/.zshrc, ~/.config/fish/config.fish
```

## Usage

### Command: `context dir`

Generate a directory tree and copy it to your clipboard.

**Examples:**
```bash
# Current directory (tree format)
context dir

# Specific directory
context dir ~/projects/myproject

# Limit depth and exclude patterns
context dir ~/projects --depth 2 --exclude "node_modules,.git"

# JSON format (great for AI analysis)
context dir --format json

# Markdown format (great for documentation)
context dir --format markdown

# Include hidden files
context dir --hidden

# Just print, don't copy to clipboard
context dir --no-copy
```

**Options:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--depth` | `-d` | Maximum depth (0 = unlimited) | `0` |
| `--exclude` | `-e` | Comma-separated patterns to exclude | `""` |
| `--hidden` | `-H` | Include hidden files | `false` |
| `--format` | `-f` | Output format: `tree`, `json`, or `markdown` | `tree` |
| `--no-copy` | `-c` | Print only, don't copy to clipboard | `false` |

### Command: `context last`

Copy recent command outputs to your clipboard. Requires shell integration setup.

**Examples:**
```bash
# Copy last command output
context last

# Copy last 3 command outputs
context last 3

# Markdown format (great for AI)
context last 5 --format markdown

# Just print, don't copy to clipboard
context last --no-copy
```

**Options:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--format` | `-f` | Output format: `raw`, `command`, or `markdown` | `raw` |
| `--raw` | `-r` | Raw output without formatting | `false` |
| `--no-copy` | `-c` | Print only, don't copy to clipboard | `false` |

### Other Commands

```bash
# Show shell integration setup instructions
context init

# Show version
context version

# Show help
context --help
context dir --help
```

## How It Works

**`context dir`**: Walks the directory tree and generates output in your chosen format (tree/JSON/markdown), then copies it to your clipboard.

**`context last`**: Shell integration hooks capture command outputs to `~/.local/share/context/history.jsonl`. The `context last` command reads this history and copies recent outputs to your clipboard.

## Troubleshooting

### `context last` says "history file not found"

You need to set up shell integration first:

1. Run `context init` to see setup instructions
2. Add the appropriate `source` command to your shell config
3. Restart your terminal or run `source ~/.bashrc` (or `~/.zshrc`)
4. Run a few commands to build up history
5. Try `context last` again

### Command not found

Make sure the installation directory is in your PATH:

```bash
# For ~/.local/bin installs
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Verify it's working
which context
```

### Clipboard not working

Context uses platform-specific clipboard tools:
- **Linux**: `xclip` or `xsel` (install via your package manager)
- **macOS**: `pbcopy` (built-in)
- **Windows**: `clip` (built-in)

### NixOS: "cannot run dynamically linked executable"

If you built with `go build`, the binary won't work on NixOS. Use one of these instead:
- `nix build` to build with Nix
- `go run ./cmd/context` to run directly
- `nix profile install` to install via Nix

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

# Run with Nix (one-off)
nix run . -- dir ~/projects
nix run github:jupiterozeye/context -- dir ~/projects
```

## Files Reference

- **`flake.nix`** - Nix flake definition for modern Nix (flakes-enabled)
- **`default.nix`** - Legacy Nix expression for `nix-build` (non-flakes). Use if you don't have flakes enabled: `nix-build -A context`
- **`shell/`** - Shell integration scripts (bash, zsh, fish)

## License

MIT