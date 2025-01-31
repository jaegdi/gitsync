#!/Am/bash

# Set the directory for the issuing files
OUTPUT_DIR="build"
mkdir -p $OUTPUT_DIR

# Build the Linux Executable
echo "Build Linux Executable..."
if ! GOOS=linux GOARCH=amd64 go build -v -o $OUTPUT_DIR/gitsync; then
    echo "Error building the Linux Executable"
    exit 1
fi
echo '----------------------------------------------------------------'

# Build the Windows Executable
echo "Build Windows Executable..."
if ! GOOS=windows GOARCH=amd64 go build -v -o $OUTPUT_DIR/gitsync.exe; then
    echo "Error building the Windows Executable"
    exit 1
fi
cp build/gitsync.exe /gast-drive-d/Daten/
echo '----------------------------------------------------------------'

echo "Build successfully completed!"
