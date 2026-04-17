$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$tag    = 'v1.0.0'
$url    = "https://github.com/Cod-e-Codes/marchat/releases/download/$tag/marchat-$tag-windows-amd64.zip"
$checksum = '1b0d7a60ef1926dbcb7e071e6a16214a48e6c743fd02e134fb1200101d3ed01b'

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
