$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$tag    = 'v1.3.1'
$url    = "https://github.com/Cod-e-Codes/marchat/releases/download/$tag/marchat-$tag-windows-amd64.zip"
$checksum = 'a2cb2601208849940befaf617842592cdfcaa0b5d5272eac13f661884ebe92db'

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
