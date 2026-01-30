#!/bin/bash
set -euo pipefail

# CMDO Installer for macOS (Apple Silicon)
# Usage: ./install-arm64.sh

BOLD='\033[1m'
SUCCESS='\033[38;2;47;191;113m'
WARN='\033[38;2;255;176;32m'
ERROR='\033[38;2;226;61;45m'
INFO='\033[38;2;255;138;91m'
NC='\033[0m'

BINARY_NAME="cmdo-macos-arm64"
INSTALL_NAME="cmdo"

echo ""
echo -e "${BOLD}🚀 CMDO Installer for macOS (Apple Silicon)${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Function to check architecture
check_architecture() {
    local arch
    arch="$(uname -m)"
    if [[ "$arch" != "arm64" ]]; then
        echo -e "${ERROR}Error: This installer is for Apple Silicon (arm64) only.${NC}"
        echo -e "${INFO}Your architecture: ${arch}${NC}"
        exit 1
    fi
}

# Function to remove macOS quarantine
remove_quarantine() {
    local file="$1"
    if command_exists xattr; then
        echo -e "${INFO}→${NC} Removing macOS quarantine attributes..."
        xattr -cr "$file" 2>/dev/null || true
        # Also remove the quarantine attribute specifically
        xattr -d com.apple.quarantine "$file" 2>/dev/null || true
    fi
}

# Function to determine best installation directory
find_install_dir() {
    local dirs=(
        "/usr/local/bin"
        "$HOME/.local/bin"
        "$HOME/bin"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" && -w "$dir" ]]; then
            echo "$dir"
            return 0
        fi
    done
    
    # Create ~/.local/bin as fallback
    mkdir -p "$HOME/.local/bin"
    echo "$HOME/.local/bin"
}

# Function to check if directory is in PATH
is_in_path() {
    local dir="$1"
    case ":$PATH:" in
        *":$dir:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Function to add directory to shell profile
add_to_shell_profile() {
    local dir="$1"
    local shell_profile=""
    
    # Detect shell and profile file
    if [[ -n "${ZSH_VERSION:-}" ]] || [[ "$SHELL" == */zsh ]]; then
        shell_profile="$HOME/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]] || [[ "$SHELL" == */bash ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            shell_profile="$HOME/.bash_profile"
        else
            shell_profile="$HOME/.bashrc"
        fi
    fi
    
    if [[ -n "$shell_profile" ]]; then
        echo ""
        echo -e "${WARN}⚠️  ${dir} is not in your PATH${NC}"
        echo -e "${INFO}Add this line to your ${shell_profile}:${NC}"
        echo ""
        echo -e "${BOLD}export PATH=\"${dir}:\$PATH\"${NC}"
        echo ""
        echo -e "${INFO}Then reload your shell:${NC}"
        echo -e "${BOLD}source ${shell_profile}${NC}"
        echo -e "${INFO}Or simply open a new terminal window.${NC}"
        echo ""
    fi
}

# Main installation function
install_cmdo() {
    # Step 1: Check architecture
    check_architecture
    
    # Step 2: Check if binary exists
    if [[ ! -f "$BINARY_NAME" ]]; then
        echo -e "${ERROR}Error: ${BINARY_NAME} not found in current directory!${NC}"
        exit 1
    fi
    
    # Step 3: Remove quarantine attributes
    remove_quarantine "$BINARY_NAME"
    
    # Step 4: Determine installation directory
    local install_dir
    install_dir="$(find_install_dir)"
    echo -e "${INFO}→${NC} Installation directory: ${install_dir}"
    
    # Step 5: Try to install
    local use_sudo=false
    if [[ "$install_dir" == "/usr/local/bin" ]]; then
        # Check if we can write without sudo
        if [[ ! -w "$install_dir" ]]; then
            use_sudo=true
            echo -e "${INFO}→${NC} Requesting administrator privileges..."
        fi
    fi
    
    # Step 6: Copy binary
    echo -e "${INFO}→${NC} Installing ${INSTALL_NAME}..."
    if [[ "$use_sudo" == "true" ]]; then
        if ! sudo cp "$BINARY_NAME" "$install_dir/$INSTALL_NAME"; then
            echo -e "${ERROR}Error: Failed to copy binary (sudo required)${NC}"
            echo -e "${INFO}Trying alternative location...${NC}"
            install_dir="$HOME/.local/bin"
            mkdir -p "$install_dir"
            cp "$BINARY_NAME" "$install_dir/$INSTALL_NAME"
        fi
        sudo chmod +x "$install_dir/$INSTALL_NAME" 2>/dev/null || chmod +x "$install_dir/$INSTALL_NAME"
    else
        cp "$BINARY_NAME" "$install_dir/$INSTALL_NAME"
        chmod +x "$install_dir/$INSTALL_NAME"
    fi
    
    # Step 7: Remove quarantine from installed binary too
    remove_quarantine "$install_dir/$INSTALL_NAME"
    
    # Step 8: Verify installation
    if [[ ! -x "$install_dir/$INSTALL_NAME" ]]; then
        echo -e "${ERROR}❌ Installation failed!${NC}"
        exit 1
    fi
    
    # Step 9: Success message
    echo ""
    echo -e "${SUCCESS}${BOLD}✅ CMDO installed successfully!${NC}"
    echo ""
    echo -e "${INFO}Installed to:${NC} ${install_dir}/${INSTALL_NAME}"
    
    # Step 10: Check PATH and warn if needed
    if ! is_in_path "$install_dir"; then
        add_to_shell_profile "$install_dir"
    fi
    
    # Step 11: Version check (if possible)
    if command_exists "$INSTALL_NAME" || [[ -x "$install_dir/$INSTALL_NAME" ]]; then
        local version
        version=$("$install_dir/$INSTALL_NAME" --version 2>/dev/null || echo "")
        if [[ -n "$version" ]]; then
            echo -e "${INFO}Version:${NC} ${version}"
        fi
    fi
    
    echo ""
    echo -e "${BOLD}Next steps:${NC}"
    echo -e "1. ${INFO}cmdo setup${NC}  - Configure CMDO"
    echo -e "2. ${INFO}cmdo --help${NC} - View available commands"
    echo ""
}

# Run installation
install_cmdo
