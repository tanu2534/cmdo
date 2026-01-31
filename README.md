# CMDO - Command Logger Tool

A cross-platform command logger tool for tracking and managing command line history.

## Installation

### Windows
```powershell
iwr -useb https://raw.githubusercontent.com/tanu2534/cmdo-release/main/install.ps1 | iex
```

### macOS/Linux
```bash
curl -fsSL https://raw.githubusercontent.com/tanu2534/cmdo-release/main/install.sh | bash
```

## Setup
After installation:
```bash
cmdo setup
```

## Usage
- `cmdo setup` - Configure shell hooks
- `cmdo serve` - Start web UI (http://localhost:8089)
- `cmdo logs` - View command history
- `cmdo cleanup` - Remove hooks

## Commands
- `cmdo --help` - Show help
- `cmdo --version` - Show version

For more information, visit: https://github.com/tanu2534/cmdo