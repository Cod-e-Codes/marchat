class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.2.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.2.0/marchat-v1.2.0-darwin-arm64.zip"
      sha256 "f78d68c22c5b6758a5b4e7356fb0d572ee4e6d8fc933d48ab2d999f1099fc842"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.2.0/marchat-v1.2.0-darwin-amd64.zip"
      sha256 "e181632d8cd6ad32084f3401b5ed5e870fc4861b6dbb3db0c5cec5d983c67531"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.2.0/marchat-v1.2.0-linux-arm64.zip"
      sha256 "727198c2795f2928b78900ff3ea273dfb5fb97242093940ee812329616128beb"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.2.0/marchat-v1.2.0-linux-amd64.zip"
      sha256 "818352b09e0b522aeb1d0d23f89adbab18e561727c9ef78bea93c90560c7d25a"
    end
  end

  def install
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "marchat-client-darwin-arm64" => "marchat-client"
        bin.install "marchat-server-darwin-arm64" => "marchat-server"
      else
        bin.install "marchat-client-darwin-amd64" => "marchat-client"
        bin.install "marchat-server-darwin-amd64" => "marchat-server"
      end
    elsif OS.linux?
      if Hardware::CPU.arm?
        bin.install "marchat-client-linux-arm64" => "marchat-client"
        bin.install "marchat-server-linux-arm64" => "marchat-server"
      else
        bin.install "marchat-client-linux-amd64" => "marchat-client"
        bin.install "marchat-server-linux-amd64" => "marchat-server"
      end
    end
  end

  test do
    ENV["MARCHAT_DOCTOR_NO_NETWORK"] = "1"
    system "#{bin}/marchat-client", "-doctor-json"
  end
end
