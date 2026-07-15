$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$tag    = 'v1.3.2'
$url    = "https://github.com/Cod-e-Codes/marchat/releases/download/$tag/marchat-$tag-windows-amd64.zip"
$checksum = 'adfd7ecf8646acf02336a12573417ade4c4b8565d83f3e427dee70f0cf8065df'

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
