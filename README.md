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

# Copy shell integration scripts
sudo mkdir -p /usr/local/share/context/shell
sudo cp shell/* /usr/local/share/context/shell/
```

**Note:** `context dir` works immediately. `context last` requires [shell integration](#setup) to capture command output.

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

### `context last` - Share recent commands (requires recording)

**Must be run inside a recorded session** (started with `context rec`).
Copies commands and output from the current terminal session to clipboard.

```bash
# Last command (copies to clipboard, no terminal output)
context last

# Last 5 commands
context last 5

# Print to terminal as well as copying
context last --print

# Markdown format
context last 10 --format markdown --print

# Detailed format with metadata
context last 3 --format detailed --print
```

**Flags:**
- `-f, --format` - Output format: `raw` (default), `markdown`, or `detailed`
- `-c, --no-copy` - Don't copy to clipboard
- `-p, --print` - Print output to terminal (default: just confirmation)

**Note:** This command only works after running `context rec`. Each terminal session is isolated.
Clipboard may not work inside the recorded session - use `--print` to see output in that case.

### `context rec` - Record session with full output

Starts a recorded shell session where all commands and their output are captured.
Uses the standard `script` command for reliable capture.

```bash
# Start recording (creates 'typescript' file in current directory)
context rec

# Run your commands...
ls -la
cat file.txt
make test

# Stop recording with 'exit'
exit

# Now get the captured commands with output
context last

# Clean up when done
rm typescript
```

**Note:** Creates a `typescript` file in the current directory. This file contains
all terminal output including commands and their results.

### `context flush` - Clear all logs

Removes all log files and typescript recordings.

```bash
# Clear everything
context flush
```

### Setup

Shell integration is **optional** and provides visual indicators when recording.

**Bash:**
```bash
echo 'source /usr/local/share/context/shell/context.bash' >> ~/.bashrc
```

**Zsh:**
```bash
echo 'source /usr/local/share/context/shell/context.zsh' >> ~/.zshrc
```

**Fish:**
```bash
echo 'source /usr/local/share/context/shell/context.fish' >> ~/.config/fish/config.fish
```

Then restart your terminal or run `source ~/.bashrc` (or `~/.zshrc`, etc.).

**What it does:**
- Shows `[rec]` in prompt when recording
- Purely cosmetic - recording works without it

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

**`context last` shows "no log directory found":**

You need to enable shell integration first:
```bash
# For Bash
echo 'source /usr/local/share/context/shell/context.bash' >> ~/.bashrc
source ~/.bashrc

# For Zsh  
echo 'source /usr/local/share/context/shell/context.zsh' >> ~/.zshrc
source ~/.zshrc
```

Then run some commands before using `context last`.

## License

MIT
