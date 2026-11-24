[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("x64", "arm64")]
  [string]$ElectronArch,

  [Parameter(Mandatory = $true)]
  [ValidateSet("amd64", "arm64")]
  [string]$GoArch
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$version = (Get-Content (Join-Path $root "package.json") -Raw | ConvertFrom-Json).version
$env:VERSION = $version

$desktopBuildOutput = Join-Path $root "releases/desktop/build"
Remove-Item -Recurse -Force $desktopBuildOutput -ErrorAction SilentlyContinue
Remove-Item -Force (Join-Path $root "releases/SHA256SUMS") -ErrorAction SilentlyContinue

function Invoke-Tool {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Command,

    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  & $Command @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "Command failed with exit code $LASTEXITCODE`: $Command $($Arguments -join ' ')"
  }
}

function Remove-DirectoryIfPresent {
  param([Parameter(Mandatory = $true)] [string]$Path)

  if (Test-Path $Path) {
    Remove-Item -Recurse -Force $Path
  }
}

$contractsGenerated = Join-Path $root "packages/contracts/generated"
Remove-DirectoryIfPresent $contractsGenerated
Invoke-Tool "yarn" @("workspace", "@lwn-simulator/contracts", "generate:bundle")
Invoke-Tool "yarn" @("workspace", "@lwn-simulator/contracts", "generate:ts")
Invoke-Tool "yarn" @("workspace", "@lwn-simulator/contracts", "generate:go")
Invoke-Tool "yarn" @("workspace", "@lwn-simulator/contracts", "generate:models")

Invoke-Tool "yarn" @("workspace", "@lwn-simulator/ui-components", "build")

Remove-DirectoryIfPresent (Join-Path $root "packages/sdk/dist")
Invoke-Tool "yarn" @("workspace", "@lwn-simulator/sdk", "build:esm")
Invoke-Tool "yarn" @("workspace", "@lwn-simulator/sdk", "build:cjs")

$env:ELECTRON_BUILD = "true"
$tsconfig = Get-Content (Join-Path $root "apps/frontend/tsconfig.json") -Raw | ConvertFrom-Json
if ($tsconfig.compilerOptions.rootDir -ne "../../") {
  throw "apps/frontend/tsconfig.json must use rootDir ../../ for the workspace build"
}

Invoke-Tool "yarn" @("workspace", "@lwn-simulator/frontend", "exec", "next", "build")

$frontendSource = Join-Path $root "apps/frontend/out"
$frontendTarget = Join-Path $root "apps/backend/internal/frontend/web"
if (-not (Test-Path (Join-Path $frontendSource "index.html"))) {
  throw "Frontend export is missing or incomplete: $frontendSource"
}

$frontendTargetParent = Split-Path -Parent $frontendTarget
$frontendStaging = Join-Path $frontendTargetParent ".frontend-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Force $frontendStaging | Out-Null
try {
  Copy-Item -Path (Join-Path $frontendSource "*") -Destination $frontendStaging -Recurse -Force
  Remove-DirectoryIfPresent $frontendTarget
  Move-Item -Path $frontendStaging -Destination $frontendTarget
} catch {
  Remove-DirectoryIfPresent $frontendStaging
  throw
}

$desktopAssets = Join-Path $root "apps/desktop/assets"
New-Item -ItemType Directory -Force $desktopAssets | Out-Null
$backendOutput = Join-Path $desktopAssets "lwn-server.exe"

Push-Location (Join-Path $root "apps/backend")
try {
  $env:GOOS = "windows"
  $env:GOARCH = $GoArch
  $env:CGO_ENABLED = "0"
  Invoke-Tool "go" @(
    "build",
    "-ldflags",
    "-X lwn-simulator-backend/version.AppVersion=$version",
    "-o",
    $backendOutput,
    "-trimpath",
    "-buildvcs=false",
    "./cmd/server/main.go"
  )
} finally {
  Pop-Location
}

Invoke-Tool "yarn" @("workspace", "@lwn-simulator/desktop-launcher", "exec", "next", "build")
Invoke-Tool "yarn" @(
  "workspace",
  "@lwn-simulator/desktop",
  "build",
  "--win",
  "nsis",
  "portable",
  "--$ElectronArch",
  "--publish",
  "never"
)

$artifacts = @(Get-ChildItem -Path $desktopBuildOutput -File -Filter "*.exe")
if (-not (Test-Path $backendOutput)) {
  throw "Windows backend was not produced"
}
if ((Get-Item $backendOutput).Length -le 0) {
  throw "Windows backend is empty"
}
if ($artifacts.Count -lt 2) {
  throw "Expected NSIS and portable Windows executables"
}
foreach ($artifact in $artifacts) {
  if ($artifact.Length -le 0) {
    throw "Windows artifact is empty: $($artifact.FullName)"
  }
}

if ($ElectronArch -eq "x64") {
  Invoke-Tool $backendOutput @("--help")
}

$checksumPath = Join-Path $root "releases/SHA256SUMS"
$checksumFiles = @($backendOutput) + @($artifacts | ForEach-Object { $_.FullName })
$checksumLines = foreach ($file in $checksumFiles) {
  $hash = (Get-FileHash -Algorithm SHA256 -Path $file).Hash.ToLowerInvariant()
  $relativePath = $file.Substring($root.Length + 1).Replace("\", "/")
  "$hash  $relativePath"
}
Set-Content -Path $checksumPath -Value $checksumLines -Encoding utf8

Write-Output "Windows artifacts"
Get-FileHash -Algorithm SHA256 -Path $checksumFiles
