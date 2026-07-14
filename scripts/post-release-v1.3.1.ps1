# Post-release packaging for v1.3.1 (run after GitHub release assets are uploaded).
# Does NOT commit or push - prints commands at the end.
#
# Usage:
#   cd marchat
#   .\scripts\post-release-v1.3.1.ps1

$ErrorActionPreference = 'Stop'
$Tag = 'v1.3.1'
$Ver = '1.3.1'
$Date = '2026-07-14'
$RepoRoot = Split-Path -Parent $PSScriptRoot
$ChocoDir = Join-Path $RepoRoot 'packaging\chocolatey'

function ConvertTo-BashPath {
    param(
        [string]$WindowsPath,
        [ValidateSet('GitBash', 'Wsl')]
        [string]$Flavor = 'GitBash'
    )
    $full = [System.IO.Path]::GetFullPath($WindowsPath)
    if ($full -notmatch '^([A-Za-z]):\\(.*)$') {
        return ($full -replace '\\', '/')
    }
    $drive = $Matches[1].ToLower()
    $rest = $Matches[2] -replace '\\', '/'
    if ($Flavor -eq 'Wsl') {
        return "/mnt/$drive/$rest"
    }
    return "/$drive/$rest"
}

function Resolve-RenderBash {
    $gitBash = @(
        "${env:ProgramFiles}\Git\bin\bash.exe",
        "${env:ProgramFiles(x86)}\Git\bin\bash.exe"
    ) | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1

    if ($gitBash) {
        return @{ Exe = $gitBash; Flavor = 'GitBash' }
    }

    $bash = Get-Command bash -ErrorAction SilentlyContinue
    if (-not $bash) {
        return $null
    }

    $flavor = 'GitBash'
    if ($bash.Source -match 'system32\\bash\.exe$') {
        $flavor = 'Wsl'
    }
    return @{ Exe = $bash.Source; Flavor = $flavor }
}

$ZipUrl = "https://github.com/Cod-e-Codes/marchat/releases/download/$Tag/marchat-$Tag-windows-amd64.zip"
$ZipPath = Join-Path $env:TEMP "marchat-$Tag-windows-amd64.zip"

Write-Host "Checking release zip..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $ZipUrl -OutFile $ZipPath -UseBasicParsing
} catch {
    throw "Release zip not found at $ZipUrl. Publish the GitHub release first."
}

$hash = (Get-FileHash -Algorithm SHA256 $ZipPath).Hash.ToLower()
Write-Host "windows-amd64 SHA256: $hash"

# Chocolatey
$installPs1 = Join-Path $ChocoDir 'tools\chocolateyinstall.ps1'
$content = Get-Content $installPs1 -Raw
$content = $content -replace "\`$tag\s*=\s*'v[^']+'", "`$tag    = '$Tag'"
$content = $content -replace "\`$checksum\s*=\s*'[0-9a-f]+'", "`$checksum = '$hash'"
Set-Content -Path $installPs1 -Value $content -NoNewline

$nuspec = Join-Path $ChocoDir 'marchat.nuspec'
$ns = Get-Content $nuspec -Raw
$ns = $ns -replace '<version>[^<]+</version>', "<version>$Ver</version>"
$ns = $ns -replace 'releases/tag/v[^<]+', "releases/tag/$Tag"
Set-Content -Path $nuspec -Value $ns -NoNewline

Push-Location $ChocoDir
choco pack
if ($LASTEXITCODE -ne 0) { throw 'choco pack failed' }
Pop-Location
Write-Host "Built $(Join-Path $ChocoDir "marchat.$Ver.nupkg")" -ForegroundColor Green

# Render other manifests (Git Bash preferred; WSL bash uses /mnt/c paths)
$bashInfo = Resolve-RenderBash
if ($bashInfo) {
    Write-Host "Running render-release-manifests.sh via $($bashInfo.Exe)..." -ForegroundColor Cyan
    $bashRepo = ConvertTo-BashPath $RepoRoot -Flavor $bashInfo.Flavor
    $bashCmd = "cd '$bashRepo' && RELEASE_TAG='$Tag' RELEASE_DATE_UTC='$Date' bash ./packaging/ci/render-release-manifests.sh"
    & $bashInfo.Exe -lc $bashCmd
    if ($LASTEXITCODE -ne 0) { throw 'render-release-manifests.sh failed' }

    Copy-Item (Join-Path $RepoRoot 'packaging-out\marchat.rb') (Join-Path $RepoRoot 'packaging\homebrew\marchat.rb') -Force
    Copy-Item (Join-Path $RepoRoot 'packaging-out\marchat.json') (Join-Path $RepoRoot 'packaging\scoop\marchat.json') -Force
    Copy-Item (Join-Path $RepoRoot 'packaging-out\aur\PKGBUILD') (Join-Path $RepoRoot 'packaging\aur\PKGBUILD') -Force
    Copy-Item (Join-Path $RepoRoot 'packaging-out\aur\.SRCINFO') (Join-Path $RepoRoot 'packaging\aur\.SRCINFO') -Force

    $wingetDest = Join-Path $RepoRoot "packaging\winget\manifests\c\Cod-e-Codes\Marchat\$Ver"
    New-Item -ItemType Directory -Force -Path $wingetDest | Out-Null
    Copy-Item (Join-Path $RepoRoot 'packaging-out\winget\*.yaml') $wingetDest -Force
    Write-Host "Synced packaging/ from packaging-out/" -ForegroundColor Green
} else {
    Write-Host "bash not found - skip render script. Run in Git Bash:" -ForegroundColor Yellow
    Write-Host "  RELEASE_TAG=$Tag RELEASE_DATE_UTC=$Date packaging/ci/render-release-manifests.sh"
}

Write-Host ""
Write-Host '=== USER: commit marchat packaging ===' -ForegroundColor Yellow
Write-Host @"
cd "$RepoRoot"
git add packaging/ release-notes-v1.3.1.md
git commit -m "chore(packaging): sync v1.3.1 release checksums"
git push origin main
"@

Write-Host ""
Write-Host '=== USER: push Chocolatey ===' -ForegroundColor Yellow
Write-Host @"
cd "$ChocoDir"
choco push marchat.$Ver.nupkg --source https://push.chocolatey.org/
"@

Write-Host ""
Write-Host '=== USER: verify packaging-forks (after CI publish-downstream-packages) ===' -ForegroundColor Yellow
Write-Host @"
cd C:\Users\Cody\Projects\GitHub\Personal\Active\packaging-forks\homebrew-marchat
git pull origin main
cd ..\scoop-marchat
git pull origin main
cd ..\winget-pkgs
git fetch origin; git checkout marchat-$Ver; git pull origin marchat-$Ver
winget validate manifests\c\Cod-e-Codes\Marchat\$Ver
gh pr list --repo microsoft/winget-pkgs --head Cod-e-Codes:marchat-$Ver
"@
