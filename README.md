# dirtop

A real-time terminal dashboard for monitoring directories. Tracks file count, directory count, lines of code, and file extension breakdown — with a live history chart that updates as your codebase evolves.

```
 ▶ dirtop  [~/code/myproject]  ↺ 1s
  files  312    │  dirs  47    │  lines  18,432
────────────────────────────────────────────────────────────────────────────────
 HISTORY  [c: area]  [m: lines]
⣀⣀⣤⣤⣶⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
t=0s────────────────────────────────────────────────────────────────────now
────────────────────────────────────────────────────────────────────────────────
 EXTENSIONS
 .go              12,104    .md               2,341
 .json             2,891    .yaml               891
 .sh                 205    (no ext)             --
```

## Features

- **Real-time scanning** — refreshes every second (configurable)
- **5 chart modes** — cycle through area, sparkline, multi-line, delta, and horizontal bar
- **3 tracked metrics** — lines of code, file count, directory count
- **Extension breakdown** — sorted by line count in a two-column grid
- **Adaptive layout** — responds to terminal resize
- **Configurable intervals** — set available refresh rates at startup

## Installation

### Homebrew (macOS & Linux)

```bash
brew tap arthurrio/dirtop
brew install dirtop
```

### One-line install (Linux & macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/arthurrio/dirtop/main/install.sh | bash
```

### Using Go

Requires Go 1.21+.

```bash
go install github.com/arthurrio/dirtop@latest
```

### Build from source

```bash
git clone https://github.com/arthurrio/dirtop.git
cd dirtop
go build -o dirtop .
sudo mv dirtop /usr/local/bin/
```

## Uninstall

### Homebrew

```bash
brew uninstall dirtop
brew untap arthurrio/dirtop
```

### Go install

```bash
rm $(which dirtop)
```

Or, if installed to the default Go bin directory:

```bash
rm ~/go/bin/dirtop
```

### Manual / install.sh

```bash
sudo rm /usr/local/bin/dirtop
```

## Usage

```
dirtop [flags] [directory]
```

If no directory is given, the current working directory is used.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dev` | `false` | Ignore common dependency, build, cache, IDE, VCS directories and generated files |
| `--ignore-dirs <list>` | `""` | Additional directory names to ignore, comma-separated |
| `-i <seconds>` | `1` | Starting refresh interval |
| `--intervals <list>` | `1,5,10,30,60` | Available intervals to cycle through (comma-separated, in seconds) |

### Examples

```bash
# Monitor current directory
dirtop

# Monitor a specific path
dirtop ~/code/myproject

# Start with a 5-second interval
dirtop -i 5

# Ignore common dependency and IDE directories
dirtop --dev

# Ignore custom directories by name
dirtop --ignore-dirs generated,coverage,tmp

# Use a custom interval list
dirtop --intervals 1,10,60

# Combine flags and path
dirtop -i 10 --intervals 1,10,60 ~/code/myproject
```

With `--dev`, `dirtop` skips common development-only paths such as `node_modules`, `vendor`, `target`, `.gradle`, `.terraform`, `__pycache__`, `.idea`, `.vscode`, `dist`, `build`, and generated files like `*.iml`, `*.class`, `*.so`, and `*.min.js.map`. Project config files such as `package.json`, `pom.xml`, `Cargo.toml`, and `tsconfig.json` are still counted.
The full directory list ignored by `--dev` is defined in [scanner.go](/home/thurrio/workspace/dirtop/scanner.go#L14).

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `c` | Cycle chart mode (area → line → multi → delta → bars) |
| `m` | Cycle active metric (lines → files → dirs) |
| `i` | Cycle refresh interval |
| `q` / `Ctrl+C` | Quit |

## Chart modes

| Mode | Description |
|------|-------------|
| `area` | Braille area chart with filled region (default) |
| `line` | Braille sparkline — line only, no fill |
| `multi` | Three overlaid sparklines: lines (purple), files (blue), dirs (green) |
| `delta` | Change between samples — growth in green, decline in orange |
| `bars` | Horizontal bar chart with time labels and values |

## Requirements

- A terminal with UTF-8 support and 256-color (or truecolor)
- Terminal width of at least 40 columns and height of at least 10 rows

## License

[MIT](LICENSE)
