# Stop on any error
$ErrorActionPreference = "Stop"

Write-Host "Setting up virtual environment for Reddit Stream Console..." -ForegroundColor Green

# Check if python is installed
try {
    python --version
} catch {
    Write-Host "Python is not installed or not in PATH. Please install Python 3.8 or higher." -ForegroundColor Red
    exit 1
}

# Check if venv exists and remove if it does
if (Test-Path "venv") {
    Write-Host "Removing existing virtual environment..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force "venv"
}

# Create new virtual environment
Write-Host "Creating new virtual environment..." -ForegroundColor Green
python -m venv venv

# Activate virtual environment
Write-Host "Activating virtual environment..." -ForegroundColor Green
.\venv\Scripts\Activate.ps1

# Upgrade pip
Write-Host "Upgrading pip..." -ForegroundColor Green
python -m pip install --upgrade pip

# Install requirements
Write-Host "Installing requirements..." -ForegroundColor Green
pip install -r requirements.txt

Write-Host "`nSetup complete! You can now run the application with:" -ForegroundColor Green
Write-Host ".\venv\Scripts\python.exe reddit_stream.py" -ForegroundColor Cyan

# Keep the window open if double-clicked
if ($Host.Name -eq "ConsoleHost") {
    Write-Host "`nPress any key to exit..."
    $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown") | Out-Null
}
