class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "0.11.0-beta.5"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.5/marchat-v0.11.0-beta.5-darwin-arm64.zip"
      sha256 "1cf8cdcb0f35f9e70fc84aa92e72caf2de3f1e0b431be033ad0d814f2234b9bf"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.5/marchat-v0.11.0-beta.5-darwin-amd64.zip"
      sha256 "4ddf24eeabcaa6214289382a81f3a310cbd645f663f37c1b3b63b9640610380b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.5/marchat-v0.11.0-beta.5-linux-arm64.zip"
      sha256 "bc7dc2048d488869d1a56af2f2b2c94c68043f2d3a0d4ba08b66893e46e1252d"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.5/marchat-v0.11.0-beta.5-linux-amd64.zip"
      sha256 "9d12a7d422723abf328c35428902507e1db79b1386c41213952c71a8d328a0ad"
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
