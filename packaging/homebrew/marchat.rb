class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.0.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-darwin-arm64.zip"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-darwin-amd64.zip"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-linux-arm64.zip"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-linux-amd64.zip"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
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
