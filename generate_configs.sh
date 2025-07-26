#!/bin/bash

# Script to generate cards.yaml and hass.yaml from config.yaml
# Usage: ./generate_configs.sh

echo "Generating configuration files from config.yaml..."

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed"
    exit 1
fi

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo "Error: config.yaml not found in current directory"
    exit 1
fi

# Run the Python script
python3 generate_configs.py

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Successfully generated:"
    echo "   - cards.yaml"
    echo "   - hass.yaml"
    echo ""
    echo "You can now use these files in your Home Assistant configuration."
else
    echo "❌ Error generating configuration files"
    exit 1
fi 