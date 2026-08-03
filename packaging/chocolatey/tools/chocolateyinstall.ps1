$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$tag    = 'v1.3.3'
$url    = "https://github.com/Cod-e-Codes/marchat/releases/download/$tag/marchat-$tag-windows-amd64.zip"
$checksum = '6d8a2cceb142f8e428f2f4b01c980ff3b1e0121d2deb6f88d1f5f011be5527af'

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
