#!/bin/bash
echo "🚀 Installing CMDO for Apple Silicon Mac..."
sudo cp cmdo-macos-arm64 /usr/local/bin/cmdo
sudo chmod +x /usr/local/bin/cmdo
echo "✅ Installation complete!"
echo "Run: cmdo setup"
