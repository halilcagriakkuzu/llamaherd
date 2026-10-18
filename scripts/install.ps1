<#
.SYNOPSIS
  LlamaHerd installer for Windows 11 (terminal).

.DESCRIPTION
  Installs LlamaHerd on Windows. Two modes:

    wsl    (default) Run the Linux/macOS installer inside a WSL distro (the
           supported path: the GUI runs on WSLg) and create Start Menu +
           Desktop shortcuts on the Windows host.

    native Download/build a native llamaherd.exe, install it under
           %LOCALAPPDATA%\LlamaHerd, add it to the user PATH, and create
           shortcuts. NOTE: native Windows support is experimental.

.PARAMETER Mode
  "wsl" (default) or "native".

.PARAMETER Distro
  WSL distro name for wsl mode (default: Ubuntu).

.PARAMETER RepoPath
  Linux path to the LlamaHerd checkout inside WSL (wsl mode). Auto-detected
  when this script is run from a \\wsl.localhost\<distro>\... path.

.PARAMETER FromRelease / FromSource
  Build from a local checkout (default when possible) or download a prebuilt
  binary from GitHub Releases.

.PARAMETER Tag / Repo
  Release tag (default: latest) and GitHub repo OWNER/REPO for downloads.

.PARAMETER Prefix
  Install directory for native mode (default: %LOCALAPPDATA%\LlamaHerd).

.PARAMETER WithDeps
  Install system dependencies without asking (wsl: apt packages).

.PARAMETER NoShortcuts
  Do not create Start Menu / Desktop shortcuts.

.PARAMETER Uninstall
  Remove a previous installation.

.PARAMETER Purge
  With -Uninstall, also remove user data + configuration.

.PARAMETER Yes
  Assume yes (non-interactive).

.EXAMPLE
  powershell -File scripts\install.ps1
  powershell -File scripts\install.ps1 -FromRelease -Repo myuser/LlamaHerd
  powershell -File scripts\install.ps1 -Uninstall -Purge
  powershell -File scripts\install.ps1 -Mode native -FromRelease
#>
[CmdletBinding()]
param(
    [ValidateSet("wsl", "native")]
    [string]$Mode = "wsl",
    [string]$Distro = "Ubuntu",
    [string]$RepoPath = "",
    [switch]$FromRelease,
    [switch]$FromSource,
    [string]$Tag = "latest",
    [string]$Repo = "",
    [string]$Prefix = "",
    [switch]$WithDeps,
    [switch]$NoShortcuts,
    [switch]$Uninstall,
    [switch]$Purge,
    [switch]$Yes
)

$ErrorActionPreference = "Stop"
$App = "llamaherd"

function Write-Step($m) { Write-Host "==> $m" -ForegroundColor Cyan }
function Write-Info($m) { Write-Host "    $m" }
function Write-Warn($m) { Write-Host "warn: $m" -ForegroundColor Yellow }
function Write-Err($m) { Write-Host "error: $m" -ForegroundColor Red }
function Have($c) { return [bool](Get-Command $c -ErrorAction SilentlyContinue) }
function Get-RepoSlug {
    if ($Repo) { return $Repo }
    if ($env:LLAMAHERD_REPO) { return $env:LLAMAHERD_REPO }
    return "halilcagriakkuzu/llamaherd"
}
function Get-InstallHost {
    if ($Prefix) { return $Prefix }
    return (Join-Path $env:LOCALAPPDATA "LlamaHerd")
}

# Convert a \\wsl.localhost\<distro>\... UNC path to a Linux path.
function ConvertFrom-WslUnc([string]$Path, [string]$DistroName) {
    foreach ($pre in @("\\wsl.localhost\$DistroName", "\\wsl$\$DistroName")) {
        if ($Path.StartsWith($pre, [System.StringComparison]::OrdinalIgnoreCase)) {
            return ($Path.Substring($pre.Length) -replace '\\', '/')
        }
    }
    return $null
}

function New-Lnk([string]$Path, [string]$Target, [string]$Arguments, [string]$Description, [string]$IconPath) {
    $dir = Split-Path -Parent $Path
    if ($dir) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
    $shell = New-Object -ComObject WScript.Shell
    $lnk = $shell.CreateShortcut($Path)
    $lnk.TargetPath = $Target
    $lnk.Arguments = $Arguments
    $lnk.WorkingDirectory = "C:\"
    $lnk.Description = $Description
    if ($IconPath -and (Test-Path $IconPath)) { $lnk.IconLocation = "$IconPath,0" }
    $lnk.Save()
    Write-Info "created $Path"
}

function Get-HostIconPath([string]$DistroName, [string]$LinuxRepo) {
    # %USERPROFILE%\LlamaHerd, not %LOCALAPPDATA%: Explorer rendered a blank icon for
    # shortcuts pointing into AppData on some machines. The file name carries a
    # content hash so a changed icon is not served from Explorer's cache.
    $iconDir = Join-Path $env:USERPROFILE "LlamaHerd"
    New-Item -ItemType Directory -Force -Path $iconDir | Out-Null
    $iconPath = Join-Path $iconDir "icon.ico"
    $linuxIcon = "\\wsl.localhost\$DistroName$($LinuxRepo -replace '/', '\')\build\windows\icon.ico"
    if (Test-Path $linuxIcon) {
        $hash = (Get-FileHash -Algorithm SHA1 $linuxIcon).Hash.Substring(0, 8).ToLower()
        $iconPath = Join-Path $iconDir "icon-$hash.ico"
        Get-ChildItem $iconDir -Filter "icon*.ico" -ErrorAction SilentlyContinue |
            Where-Object { $_.FullName -ne $iconPath } | Remove-Item -Force -ErrorAction SilentlyContinue
        Copy-Item -Force $linuxIcon $iconPath
    }
    return $iconPath
}

function Add-UserPath([string]$Dir) {
    $cur = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $cur) { $cur = "" }
    $parts = $cur -split ';' | Where-Object { $_ }
    if ($parts -notcontains $Dir) {
        [Environment]::SetEnvironmentVariable("Path", (($parts + $Dir) -join ';'), "User")
        Write-Info "added $Dir to the user PATH (restart your terminal)"
    }
}
function Remove-UserPath([string]$Dir) {
    $cur = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $cur) { return }
    $parts = $cur -split ';' | Where-Object { $_ -and ($_.TrimEnd('\') -ne $Dir.TrimEnd('\')) }
    [Environment]::SetEnvironmentVariable("Path", ($parts -join ';'), "User")
}

# ---------------------------------------------------------------------------
# WSL mode
# ---------------------------------------------------------------------------
function Install-Wsl {
    if (-not (Have "wsl.exe")) { throw "wsl.exe not found. Install WSL: 'wsl --install'." }

    $linuxRepo = $RepoPath
    if (-not $linuxRepo) {
        $repoWin = Split-Path -Parent $PSScriptRoot
        $linuxRepo = ConvertFrom-WslUnc $repoWin $Distro
    }
    if (-not $linuxRepo) {
        throw "cannot determine the Linux repo path; pass -RepoPath /home/<you>/.../LlamaHerd"
    }
    Write-Step "LlamaHerd installer (WSL: $Distro)"
    Write-Info "repo: $linuxRepo"

    $installArgs = @()
    if ($FromSource) { $installArgs += "--from-source" }
    if ($FromRelease) { $installArgs += "--from-release" }
    if ($FromRelease) { $installArgs += @("--release-tag", $Tag) }
    if ($Repo) { $installArgs += @("--repo", $Repo) }
    if ($WithDeps) { $installArgs += "--with-deps" }
    if ($Yes) { $installArgs += "--yes" }
    if ($Uninstall) { $installArgs += "--uninstall" }
    if ($Purge) { $installArgs += "--purge" }

    $inner = "cd '$linuxRepo' && bash scripts/install.sh " + ($installArgs -join ' ')
    & wsl.exe -d $Distro -- bash -lc $inner
    if ($LASTEXITCODE -ne 0) { throw "install.sh failed inside WSL (exit $LASTEXITCODE)" }

    if ($Uninstall) {
        Remove-Shortcuts
        return
    }
    if (-not $NoShortcuts) { New-WslShortcuts $Distro $linuxRepo }
}

function New-WslShortcuts([string]$DistroName, [string]$LinuxRepo) {
    $wslg = "C:\Program Files\WSL\wslg.exe"
    if (-not (Test-Path $wslg)) { $wslg = "$env:SystemRoot\System32\wsl.exe" }
    $icon = Get-HostIconPath $DistroName $LinuxRepo
    $launcher = "$LinuxRepo/scripts/launch-gui.sh"
    $arguments = "-d $DistroName --cd `"$LinuxRepo`" -- /bin/bash `"$launcher`""
    $desc = "LlamaHerd - llama.cpp control panel (WSL: $DistroName)"
    Write-Step "Creating shortcuts"
    New-Lnk (Join-Path ([Environment]::GetFolderPath("Desktop")) "LlamaHerd.lnk") $wslg $arguments $desc $icon
    New-Lnk (Join-Path ([Environment]::GetFolderPath("Programs")) "LlamaHerd.lnk") $wslg $arguments $desc $icon
}

function Remove-Shortcuts {
    Write-Step "Removing shortcuts"
    foreach ($p in @(
            (Join-Path ([Environment]::GetFolderPath("Desktop")) "LlamaHerd.lnk"),
            (Join-Path ([Environment]::GetFolderPath("Programs")) "LlamaHerd.lnk"))) {
        if (Test-Path $p) { Remove-Item -Force $p; Write-Info "removed $p" }
    }
}

# ---------------------------------------------------------------------------
# native mode (experimental)
# ---------------------------------------------------------------------------
function Install-Native {
    $prefix = Get-InstallHost
    $binDir = Join-Path $prefix "bin"
    Write-Step "LlamaHerd installer (native Windows)"
    Write-Info "prefix: $prefix"

    if ($Uninstall) {
        Write-Step "Uninstalling"
        if (Test-Path $prefix) { Remove-Item -Recurse -Force $prefix; Write-Info "removed $prefix" }
        Remove-UserPath $binDir
        if (-not $NoShortcuts) { Remove-Shortcuts }
        if ($Purge) {
            $data = Join-Path $env:LOCALAPPDATA "llamaherd"
            $cfg = Join-Path $env:APPDATA "llamaherd"
            foreach ($d in @($data, $cfg)) {
                if (Test-Path $d) { Remove-Item -Recurse -Force $d; Write-Info "removed $d" }
            }
        }
        return
    }

    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    $exe = Join-Path $binDir "llamaherd.exe"

    if ($FromSource) {
        if (-not (Have "go")) { throw "Go toolchain not found (needed for -FromSource)" }
        Write-Step "Building from source"
        Push-Location (Split-Path -Parent $PSScriptRoot)
        try {
            & go build -tags "desktop,production" -o $exe .
            if ($LASTEXITCODE -ne 0) { throw "go build failed" }
        }
        finally { Pop-Location }
    }
    else {
        $slug = Get-RepoSlug
        $asset = "llamaherd_windows_amd64.zip"
        if ($Tag -eq "latest") {
            $url = "https://github.com/$slug/releases/latest/download/$asset"
        }
        else {
            $url = "https://github.com/$slug/releases/download/$Tag/$asset"
        }
        Write-Step "Downloading $url"
        $tmp = Join-Path $env:TEMP ("llamaherd-" + [guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Force -Path $tmp | Out-Null
        $zip = Join-Path $tmp $asset
        Invoke-WebRequest -Uri $url -OutFile $zip
        Expand-Archive -Path $zip -DestinationPath $tmp -Force
        $found = Get-ChildItem -Path $tmp -Recurse -Filter "llamaherd.exe" | Select-Object -First 1
        if (-not $found) { throw "llamaherd.exe not found in $asset" }
        Copy-Item -Force $found.FullName $exe
    }
    Write-Info "installed $exe"

    Add-UserPath $binDir
    if (-not $NoShortcuts) {
        Write-Step "Creating shortcuts"
        New-Lnk (Join-Path ([Environment]::GetFolderPath("Desktop")) "LlamaHerd.lnk") $exe "" "LlamaHerd - llama.cpp control panel" ""
        New-Lnk (Join-Path ([Environment]::GetFolderPath("Programs")) "LlamaHerd.lnk") $exe "" "LlamaHerd - llama.cpp control panel" ""
    }
    Write-Info "Config: %APPDATA%\llamaherd\config.json"
}

# ---------------------------------------------------------------------------
# main
# ---------------------------------------------------------------------------
if ($Mode -eq "wsl") { Install-Wsl } else { Install-Native }
Write-Step "Done."
Write-Host ""
Write-Info "Uninstall: powershell -File scripts\install.ps1 -Mode $Mode -Uninstall"
