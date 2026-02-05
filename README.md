# Context

Terminal context capture tool for AI-assisted debugging. Simplifies sharing terminal context with AI without manual copy-pasting.

## Features

- **`context dir [path]`** - Generate directory tree and copy to clipboard
- **`context last [n]`** - Copy last n terminal outputs to clipboard
- Cross-platform clipboard support (Linux, macOS, Windows)
- Shell integration for Bash, Zsh, and Fish
- Nix flake support

## Installation

### Using Nix (Recommended)

```bash
# Build and install to your profile
nix profile install github:jupiterozeye/context

# Or run without installing (one-off usage)
nix run github:jupiterozeye/context -- dir ~/projects
nix run github:jupiterozeye/context -- last 3
```

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

### From Source

```bash
git clone https://github.com/jupiterozeye/context.git
cd context
go build -o context ./cmd/context

# Option 1: Install to /usr/local/bin (traditional Linux)
sudo mkdir -p /usr/local/bin /usr/local/share/context/shell
sudo cp context /usr/local/bin/
sudo cp -r shell/* /usr/local/share/context/shell/

# Option 2: Install to ~/.local/bin (recommended for NixOS and user installs)
mkdir -p ~/.local/bin ~/.local/share/context/shell
cp context ~/.local/bin/
cp -r shell/* ~/.local/share/context/shell/

# Make sure ~/.local/bin is in your PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

**Note for NixOS users:** NixOS doesn't use `/usr/local`. Use Option 2 (`~/.local/bin`) or prefer the Nix/NixOS installation methods above.

## Setup

### Shell Integration (Required for `context last`)

The shell integration captures command output so `context last` can access it. Choose the appropriate path based on your installation method:

**For Nix profile installs** (`nix profile install`):
```bash
# Bash (~/.bashrc)
source ~/.nix-profile/share/context/shell/context.bash

# Zsh (~/.zshrc)
source ~/.nix-profile/share/context/shell/context.zsh

# Fish (~/.config/fish/config.fish)
source ~/.nix-profile/share/context/shell/context.fish
```

**For system-wide installs** (`make install` or manual to `/usr/local`):
```bash
# Bash (~/.bashrc)
source /usr/local/share/context/shell/context.bash

# Zsh (~/.zshrc)
source /usr/local/share/context/shell/context.zsh

# Fish (~/.config/fish/config.fish)
source /usr/local/share/context/shell/context.fish
```

**For local installs** (manual to `~/.local`):
```bash
# Bash (~/.bashrc)
source ~/.local/share/context/shell/context.bash

# Zsh (~/.zshrc)
source ~/.local/share/context/shell/context.zsh

# Fish (~/.config/fish/config.fish)
source ~/.local/share/context/shell/context.fish
```

**For local development** (from repo):
```bash
# Bash (~/.bashrc)
source /path/to/context/shell/context.bash
```

Then restart your terminal or run:
```bash
source ~/.bashrc  # or ~/.zshrc, etc.
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

# One-off with nix (no install needed)
nix run github:jupiterozeye/context -- dir ~/projects
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

# One-off with nix (requires shell setup first)
nix run github:jupiterozeye/context -- last 3
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