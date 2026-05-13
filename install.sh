#!/bin/bash

# Exit on error
set -e

echo "==============================================="
echo "  AI Radio FM - Dependency Installer (macOS)   "
echo "==============================================="
echo ""

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    echo "Homebrew is not installed. Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
    echo "Homebrew is already installed. Updating..."
    brew update
fi

echo ""
echo "Installing system dependencies (Go, Icecast, Python 3.11+)..."
# Using brew to install go, icecast, and python
brew install go icecast python@3.11 uv

echo ""
echo "Setting up Python environment for Kokoro TTS..."
# Create a virtual environment specifically for TTS
if [ ! -d "venv" ]; then
    uv venv
fi

echo "Activating virtual environment and installing python dependencies..."
# Source the virtual environment
source venv/bin/activate

# Install kokoro and soundfile via uv
uv pip install kokoro soundfile

echo ""
echo "Building the AI Radio FM Go binary..."
go build -o airadio .

echo ""
echo "==============================================="
echo "  Installation Complete!                       "
echo "==============================================="
echo ""
echo "Next steps:"
echo "1. Configure your shows in config/schedule.yaml"
echo "2. Configure your hosts in config/personas.yaml"
echo "3. Start your Icecast server (e.g., 'icecast -c /usr/local/etc/icecast.xml')"
echo "4. Run AI Radio FM using: ./airadio start"
echo ""
