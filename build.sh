#!/bin/bash

# Setze das Verzeichnis für die Ausgabedateien
OUTPUT_DIR="build"
mkdir -p $OUTPUT_DIR

# Baue die Linux-Executable
echo "Baue Linux-Executable..."
if ! GOOS=linux GOARCH=amd64 go build -v -o $OUTPUT_DIR/gitsync; then
    echo "Fehler beim Bauen der Linux-Executable"
    exit 1
fi
echo '----------------------------------------------------------------'

# Baue die Windows-Executable
echo "Baue Windows-Executable..."
if ! GOOS=windows GOARCH=amd64 go build -v -o $OUTPUT_DIR/gitsync.exe; then
    echo "Fehler beim Bauen der Windows-Executable"
    exit 1
fi
cp build/gitsync.exe /gast-drive-d/Daten/
echo '----------------------------------------------------------------'

echo "Build erfolgreich abgeschlossen!"
