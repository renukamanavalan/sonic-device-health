#!/bin/bash

# This script is the entry point for the installation process.

# Get the directory where this script is located
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Check if "do-install.py" exists in the same directory as this script
if [ ! -f "$script_dir/do-install.py" ]; then
    echo "Error: do-install.py does not exist in the same directory as this script."
    exit 1
fi

# Check if Python is available
if ! command -v python &> /dev/null; then
    echo "Error: Python could not be found. Exiting installation."
    exit 1
fi

# Check if an argument was passed
if [ $# -ne 1 ]; then
    echo "Error: No argument was passed to the installation script."
    exit 1
fi

# Run the "do-install.py" script with the argument
python "$script_dir/do-install.py" "$1"
exit_status=$?

# Check the exit status of the script
if [ $exit_status -eq 0 ]; then
    echo "Installation Script executed successfully."
    exit 0
else
    echo "Error: Installation Script encountered an issue with exit code $exit_status."
    exit 1
fi
