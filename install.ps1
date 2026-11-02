# Installs O'So AI Relay for the current account, on Windows:
#
#   irm https://github.com/novalo-dev/Somcp-Relay/releases/latest/download/install.ps1 | iex
#
# It downloads the archive for this computer from the latest release, checks
# it against the release's SHA256SUMS.txt, and lets oso copy itself onto the
# PATH. Nothing needs administrator rights. The script is attached to each
# release, so the line above always runs a published version of it.
#
# It runs inside the user's window through iex: errors are thrown, never
# "exit", which would close that window.

$ErrorActionPreference = 'Stop'
# Invoke-WebRequest is many times slower while it draws its progress bar.
$ProgressPreference = 'SilentlyContinue'
# Windows PowerShell 5.1 may still default to TLS 1.0, which GitHub refuses.
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$base = 'https://github.com/novalo-dev/Somcp-Relay/releases/latest/download'

# A 32-bit PowerShell on a 64-bit Windows reports x86 here; the real
# processor is then in PROCESSOR_ARCHITEW6432.
$arch = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) { $arch = $env:PROCESSOR_ARCHITEW6432 }
$file = 'oso-ai-relay-windows-x64.zip'
if ($arch -eq 'ARM64') { $file = 'oso-ai-relay-windows-arm64.zip' }

$tmp = Join-Path ([IO.Path]::GetTempPath()) ('oso-' + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $file..."
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$file" -OutFile (Join-Path $tmp $file)
    Invoke-WebRequest -UseBasicParsing -Uri "$base/SHA256SUMS.txt" -OutFile (Join-Path $tmp 'SHA256SUMS.txt')

    # A truncated or altered download must never be run.
    $line = Get-Content (Join-Path $tmp 'SHA256SUMS.txt') | Where-Object { $_ -match ('\s' + [regex]::Escape($file) + '$') } | Select-Object -First 1
    $expected = if ($line) { ($line -split '\s+')[0].ToLower() } else { '' }
    $actual = (Get-FileHash -Algorithm SHA256 -Path (Join-Path $tmp $file)).Hash.ToLower()
    if (-not $expected -or $expected -ne $actual) {
        throw "O'So AI Relay: the download does not match the release checksum"
    }

    Expand-Archive -Path (Join-Path $tmp $file) -DestinationPath $tmp -Force

    # oso copies itself to %LOCALAPPDATA%\Programs\oso and adds that folder
    # to the user's Path.
    & (Join-Path $tmp 'oso.exe') addtopath
    if ($LASTEXITCODE -ne 0) { throw "O'So AI Relay: oso addtopath failed" }

    # The Path change reaches new windows only; this one gets it too, so
    # oso works right away.
    $dir = Join-Path $env:LOCALAPPDATA 'Programs\oso'
    if (($env:Path -split ';') -notcontains $dir) { $env:Path = "$env:Path;$dir" }

    Write-Host 'This window can run it too: oso ui'
}
finally {
    Remove-Item -Recurse -Force -Path $tmp -ErrorAction SilentlyContinue
}
