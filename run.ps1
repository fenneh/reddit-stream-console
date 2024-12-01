# Activate virtual environment and run the application
Write-Host "Starting Reddit Stream Console..." -ForegroundColor Green
.\venv\Scripts\Activate.ps1
python reddit_stream.py

# Keep the window open if double-clicked
if ($Host.Name -eq "ConsoleHost") {
    Write-Host "`nPress any key to exit..."
    $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown") | Out-Null
}
