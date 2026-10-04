# Install vpn CLI (+ sing-box core) from GitHub releases.
# Usage: irm https://raw.githubusercontent.com/SabirDzh/VpnCLI/master/install.ps1 | iex
# Run in an elevated (Administrator) PowerShell for TUN support.
$ErrorActionPreference = "Stop"
$Repo = "SabirDzh/VpnCLI"
$Bin = "vpn"
$MinSingBox = if ($env:MIN_SINGBOX) { $env:MIN_SINGBOX } else { "1.14.0" }
$SkipSingBox = $env:SKIP_SINGBOX -eq "1"

function Find-SingBoxVersion {
  $cmd = Get-Command sing-box -ErrorAction SilentlyContinue
  if (-not $cmd) { return $null }
  $first = (& sing-box version 2>$null | Select-Object -First 1)
  if ($first -match "(\d+\.\d+\.\d+)") { return $matches[1] }
  return "0.0.0"
}

function Install-SingBoxManaged {
  # winget is built into Windows 10/11.
  $w = Get-Command winget -ErrorAction SilentlyContinue
  if ($w) {
    try {
      winget install -e --id SagerNet.sing-box --accept-source-agreements --accept-package-agreements
      return $true
    } catch { Write-Host "winget failed, trying next source..." }
  }
  if (Get-Command choco -ErrorAction SilentlyContinue) {
    try { choco install sing-box -y; return $true } catch { Write-Host "choco failed, trying next source..." }
  }
  if (Get-Command scoop -ErrorAction SilentlyContinue) {
    try { scoop install sing-box; return $true } catch { Write-Host "scoop failed, trying manual download..." }
  }
  return $false
}

function Install-SingBoxManual {
  param([string]$DestDir)
  # Last resort: sing-box + wintun straight from upstream releases.
  $rel = Invoke-RestMethod "https://api.github.com/repos/SagerNet/sing-box/releases/latest"
  $tag = $rel.tag_name
  $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
  $tmp = Join-Path $env:TEMP "singbox-install"
  New-Item -ItemType Directory -Force -Path $tmp | Out-Null
  try {
    $sb = Join-Path $tmp "sing-box.zip"
    Invoke-WebRequest "https://github.com/SagerNet/sing-box/releases/download/$tag/sing-box-$($tag.TrimStart('v'))-windows-$arch.zip" -OutFile $sb
    Expand-Archive -Path $sb -DestinationPath $tmp -Force
    $exe = Get-ChildItem -Path $tmp -Recurse -Filter "sing-box.exe" | Select-Object -First 1
    if (-not $exe) { throw "sing-box.exe not found in upstream archive" }
    # Copy the whole folder (exe plus companion DLLs like libcronet.dll).
    Copy-Item (Join-Path $exe.Directory.FullName "*") $DestDir -Force
    # TUN driver required next to sing-box.exe.
    $wintun = Join-Path $tmp "wintun.zip"
    Invoke-WebRequest "https://www.wintun.net/builds/wintun-0.14.1.zip" -OutFile $wintun
    Expand-Archive -Path $wintun -DestinationPath $tmp -Force
    $dll = Get-ChildItem -Path $tmp -Recurse -Filter "wintun.dll" |
      Where-Object { $_.FullName -match $arch } | Select-Object -First 1
    if ($dll) { Copy-Item $dll.FullName (Join-Path $DestDir "wintun.dll") -Force }
  } finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
  }
}

function Ensure-SingBox {
  param([string]$DestDir)
  if ($SkipSingBox) { Write-Host "skipping sing-box setup (SKIP_SINGBOX=1)"; return }
  $ver = Find-SingBoxVersion
  if ($ver -and ([version]$ver -ge [version]$MinSingBox)) {
    Write-Host "sing-box $ver already installed (>= $MinSingBox)"
    return
  }
  if ($ver) { Write-Host "sing-box $ver is too old (need >= $MinSingBox), upgrading..." }
  else { Write-Host "installing sing-box core (>= $MinSingBox)..." }
  $managed = Install-SingBoxManaged
  if (-not $managed) { Install-SingBoxManual -DestDir $DestDir }
  $ver = Find-SingBoxVersion
  if ((-not $ver) -or ([version]$ver -lt [version]$MinSingBox)) {
    # Managed installers may not refresh PATH in this session.
    if ($managed) {
      Write-Host "sing-box installed but not yet on PATH — restart the shell, then re-run this script"
      return
    }
    # Manual install went to DestDir which may not be on PATH yet.
    $local = Join-Path $DestDir "sing-box.exe"
    if (Test-Path $local) {
      Write-Host "sing-box placed at $local (add $DestDir to PATH or run from there)"
      return
    }
    throw "sing-box install failed"
  }
  Write-Host "sing-box $ver ready"
}

$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

if ($env:VPN_VERSION) { $Tag = $env:VPN_VERSION } else {
  $latest = Invoke-WebRequest -UseBasicParsing "https://github.com/$Repo/releases/latest"
  $Tag = $latest.BaseResponse.ResponseUri.Segments[-1].Trim("/")
  if (-not $Tag) { throw "cannot resolve latest release" }
}

$dest = Join-Path ($env:ProgramFiles) "vpn"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Ensure-SingBox -DestDir $dest

$tmp = Join-Path $env:TEMP "vpn-install"
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
  $Ver = $Tag.TrimStart("v") # release assets use the version without leading v
  Write-Host "installing $Bin $Tag (Windows/$Arch)..."
  $zip = Join-Path $tmp "vpn.zip"
  Invoke-WebRequest "https://github.com/$Repo/releases/download/$Tag/${Bin}_${Ver}_windows_${Arch}.zip" -OutFile $zip
  Expand-Archive -Path $zip -DestinationPath $tmp -Force
  Copy-Item (Join-Path $tmp "$Bin.exe") (Join-Path $dest "$Bin.exe") -Force
  Write-Host "installed to $dest\$Bin.exe"
  Write-Host "Add $dest to PATH; run PowerShell as Administrator for TUN support."
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
