$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$tag    = 'v0.11.0-beta.5'
$url    = "https://github.com/Cod-e-Codes/marchat/releases/download/$tag/marchat-$tag-windows-amd64.zip"
$checksum = 'f38696037c3927c3f7c80e778f73a0af094a22f3b1fb93082495f8857948bfb4'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url           = $url
  checksum      = $checksum
  checksumType  = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

Install-BinFile -Name 'marchat-client' -Path (Join-Path $toolsDir 'marchat-client-windows-amd64.exe')
Install-BinFile -Name 'marchat-server' -Path (Join-Path $toolsDir 'marchat-server-windows-amd64.exe')
