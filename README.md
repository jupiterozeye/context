# Context

A simple CLI tool to capture and share terminal context with AI assistants.

Stop copy-pasting directory trees and command history manually - `context` does it for you and copies everything to your clipboard.

## Installation

**One-liner install (Nix with flakes):**
```bash
nix profile install github:jupiterozeye/context
```

**Other methods:**
```bash
# Try without installing
nix run github:jupiterozeye/context -- dir

# Install from source (requires Go)
git clone https://github.com/jupiterozeye/context.git
cd context
go build -o context ./cmd/context
sudo cp context /usr/local/bin/
```

That's it! No setup, no configuration files, no shell integration needed.

## Usage

### `context dir` - Share your project structure

```bash
# Current directory
context dir

# Specific directory
context dir ~/my-project

# Options
context dir --depth 2 --exclude "node_modules,.git"
context dir --format json
context dir --format markdown
context dir --hidden
context dir --no-copy  # Just print, don't copy to clipboard
```

**Flags:**
- `-d, --depth N` - Limit depth (0 = unlimited)
- `-e, --exclude` - Exclude patterns (comma-separated)
- `-f, --format` - Output format: `tree` (default), `json`, or `markdown`
- `-H, --hidden` - Include hidden files
- `-c, --no-copy` - Print only, don't copy

### `context last` - Share recent commands

```bash
# Last command
context last

# Last 5 commands  
context last 5

# Markdown format
context last 10 --format markdown
```

Reads from your shell history (`~/.zsh_history` or `~/.bash_history`).

**Flags:**
- `-f, --format` - Output format: `raw`, `command` (default), or `markdown`
- `-r, --raw` - Raw output without formatting
- `-c, --no-copy` - Print only, don't copy

## Examples

**Share your project with AI:**
```bash
cd ~/my-project
context dir
# Paste into ChatGPT/Claude: "Here's my project structure: [Ctrl+V]"
```

**Share what you just tried:**
```bash
context last 3
# Paste into AI: "I tried these commands: [Ctrl+V]"
```

**Combine both:**
```bash
context dir --format markdown > project.md
context last 10 --format markdown >> project.md
# Now project.md has everything for your AI
```

## Why Context?

Before:
1. Run `ls -R` or `tree`
2. Select and copy output
3. Paste into AI chat
4. Scroll up to find error messages
5. Select and copy those too
6. Paste again...

After:
1. `context dir` → Everything in clipboard
2. `context last 5` → Recent commands in clipboard  
3. Paste into AI ✨

## Development

```bash
# Run locally
go run ./cmd/context dir

# Build
go build -o context ./cmd/context

# Build with Nix
nix build
./result/bin/context dir
```

## NixOS Integration

Add to your NixOS `flake.nix`:

```nix
{
  inputs = {
    context.url = "github:jupiterozeye/context";
    # ... other inputs
  };
  
  # Then add to your packages:
  environment.systemPackages = [
    inputs.context.packages.${pkgs.system}.default
  ];
}
```

Or via home-manager:
```nix
{
  home.packages = [
    inputs.context.packages.${pkgs.system}.default
  ];
}
```

## Troubleshooting

**Clipboard not working:**

Install clipboard tools:
- **Linux/Wayland**: `wl-clipboard`
- **Linux/X11**: `xclip`
- **macOS**: Built-in (pbcopy)
- **Windows**: Built-in (clip)

```bash
# NixOS
nix-env -iA nixpkgs.wl-clipboard

# Ubuntu/Debian
sudo apt install wl-clipboard

# Arch
sudo pacman -S wl-clipboard
```

**`context last` shows "no history found":**

Make sure you're using bash or zsh and have a history file:
```bash
ls -la ~/.zsh_history ~/.bash_history
```

If empty, your shell might not be saving history. Check your shell config (`~/.zshrc` or `~/.bashrc`).

## License

MIT
