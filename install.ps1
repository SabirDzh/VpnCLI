# Install vpn CLI from GitHub releases (run as Administrator for TUN).
# Usage: irm https://raw.githubusercontent.com/SabirDzh/VpnCLI/master/install.ps1 | iex
$ErrorActionPreference = "Stop"
$Repo = "SabirDzh/VpnCLI"
$Bin = "vpn"

$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
  if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else { throw "32-bit Windows is not supported" }

if ($env:VPN_VERSION) { $Tag = $env:VPN_VERSION } else {
  $latest = Invoke-WebRequest -UseBasicParsing "https://github.com/$Repo/releases/latest"
  $Tag = $latest.BaseResponse.ResponseUri.Segments[-1].Trim("/")
  if (-not $Tag) { throw "cannot resolve latest release" }
}

$tmp = Join-Path $env:TEMP "vpn-install"
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
  Write-Host "installing $Bin $Tag (Windows/$Arch)..."
  $Ver = $Tag.TrimStart("v") # release assets use the version without leading v
  $zip = Join-Path $tmp "vpn.zip"
  Invoke-WebRequest "https://github.com/$Repo/releases/download/$Tag/${Bin}_${Ver}_windows_${Arch}.zip" -OutFile $zip
  Expand-Archive -Path $zip -DestinationPath $tmp -Force
  $dest = Join-Path ($env:ProgramFiles) "vpn"
  New-Item -ItemType Directory -Force -Path $dest | Out-Null
  Copy-Item (Join-Path $tmp "$Bin.exe") (Join-Path $dest "$Bin.exe") -Force
  Write-Host "installed to $dest\$Bin.exe (add to PATH or run with full path, as Administrator for TUN)"
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
