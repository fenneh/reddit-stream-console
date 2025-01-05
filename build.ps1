# PowerShell script for setting up Reddit Stream Console

Write-Host "🎭 Setting up Reddit Stream Console..."

# Check if Python is installed
try {
    $pythonVersion = python --version
    Write-Host "✓ Found Python: $pythonVersion"
}
catch {
    Write-Host "❌ Python is not installed or not in PATH. Please install Python 3 and try again."
    exit 1
}

# Check if pip is installed
try {
    $pipVersion = pip --version
    Write-Host "✓ Found pip: $pipVersion"
}
catch {
    Write-Host "❌ pip is not installed or not in PATH. Please install pip and try again."
    exit 1
}

# Create virtual environment if it doesn't exist
if (-not (Test-Path "venv")) {
    Write-Host "🔨 Creating virtual environment..."
    python -m venv venv
}

# Activate virtual environment
Write-Host "🔌 Activating virtual environment..."
.\venv\Scripts\Activate.ps1

# Install requirements
Write-Host "📦 Installing dependencies..."
pip install -r requirements.txt

# Create .env if it doesn't exist
if (-not (Test-Path ".env")) {
    Write-Host "📝 Creating .env file from template..."
    Copy-Item .env.example .env
    Write-Host "⚠️ Don't forget to edit .env with your Reddit API credentials!"
}

Write-Host "`n✨ Setup complete! To run the app:"
Write-Host "1. Make sure you're in the virtual environment (should see (venv) in your prompt)"
Write-Host "2. If not, run: .\venv\Scripts\Activate.ps1"
Write-Host "3. Then run: python main.py" 