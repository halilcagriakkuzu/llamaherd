# Create Windows 11 shortcuts that launch the LlamaHerd GUI (WSLg) from the host.
#
# Run from Windows PowerShell, or from WSL via:
#   powershell.exe -NoProfile -ExecutionPolicy Bypass -File \
#     "\\wsl.localhost\Ubuntu\home\<user>\workspace\LlamaHerd\scripts\make-windows-shortcut.ps1"
#
# Creates "LlamaHerd" on the Desktop and in the Start Menu.

param(
    [string]$Distro   = "Ubuntu",
    [string]$RepoPath = "",
    [string]$WslgExe  = "C:\Program Files\WSL\wslg.exe"
)

$ErrorActionPreference = "Stop"

# Auto-detect the Linux repo path when this script is run from a
# \\wsl.localhost\<distro>\... checkout (scripts\ is one level below the repo).
if (-not $RepoPath) {
    $root = Split-Path -Parent $PSScriptRoot
    foreach ($pre in @("\\wsl.localhost\$Distro", "\\wsl$\$Distro")) {
        if ($root.StartsWith($pre, [System.StringComparison]::OrdinalIgnoreCase)) {
            $RepoPath = ($root.Substring($pre.Length) -replace '\\', '/')
        }
    }
    if (-not $RepoPath) { throw "Pass -RepoPath <linux path of the LlamaHerd checkout> (e.g. /home/you/LlamaHerd)." }
}

if (-not (Test-Path $WslgExe)) {
    # Fall back to wsl.exe if wslg.exe is not present (an extra console window
    # will be visible in that case).
    $WslgExe = "$env:SystemRoot\System32\wsl.exe"
    Write-Warning "wslg.exe not found; falling back to $WslgExe"
}

# --- icon: copy into the Windows profile so the .lnk can reference a drive path
$iconDir  = Join-Path $env:LOCALAPPDATA "LlamaHerd"
New-Item -ItemType Directory -Force -Path $iconDir | Out-Null
$iconPath = Join-Path $iconDir "icon.ico"

$linuxIconSource = "\\wsl.localhost\$Distro$($RepoPath -replace '/', '\')\build\windows\icon.ico"
if (Test-Path $linuxIconSource) {
    Copy-Item -Force $linuxIconSource $iconPath
} else {
    Write-Warning "icon not found at $linuxIconSource (run scripts/make-icon.py first)"
}

# --- shortcut target / arguments
$launcher  = "$RepoPath/scripts/launch-gui.sh"
$arguments = "-d $Distro --cd `"$RepoPath`" -- /bin/bash `"$launcher`""
$workdir   = "C:\"

function New-LlmShortcut([string]$Path) {
    $dir = Split-Path -Parent $Path
    if ($dir) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }

    $shell = New-Object -ComObject WScript.Shell
    $lnk = $shell.CreateShortcut($Path)
    $lnk.TargetPath       = $WslgExe
    $lnk.Arguments        = $arguments
    $lnk.WorkingDirectory = $workdir
    $lnk.Description      = "LlamaHerd - llama.cpp control panel (WSL: $Distro)"
    if (Test-Path $iconPath) { $lnk.IconLocation = "$iconPath,0" }
    $lnk.Save()  # writes the .lnk

    Write-Host "created $Path"
}

New-LlmShortcut (Join-Path ([Environment]::GetFolderPath("Desktop")) "LlamaHerd.lnk")
New-LlmShortcut (Join-Path ([Environment]::GetFolderPath("Programs")) "LlamaHerd.lnk")

Write-Host ""
Write-Host "Done. Target : $WslgExe"
Write-Host "Args         : $arguments"
Write-Host "Icon         : $iconPath"
