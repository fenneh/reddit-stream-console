# PowerShell script for building Reddit Stream Console

Write-Host "Building Reddit Stream Console..."

try {
    $goVersion = go version
    Write-Host "✓ Found Go: $goVersion"
}
catch {
    Write-Host "Go is not installed or not in PATH. Please install Go 1.22+ and try again."
    exit 1
}

if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

Write-Host "Compiling binary..."
go build -o bin\reddit-stream-console.exe .\cmd\reddit-stream-console

Write-Host "`nBuild complete! Run .\bin\reddit-stream-console.exe to start."
