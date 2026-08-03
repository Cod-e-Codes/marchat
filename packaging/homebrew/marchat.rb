class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.4/marchat-v1.3.4-darwin-arm64.zip"
      sha256 "16550d188a110f38a591b9ad06376b9225e845bcdec8d8b4ec3298cf28a626fa"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.4/marchat-v1.3.4-darwin-amd64.zip"
      sha256 "c84a3e42d185d7933beefb44eeba8a57348e86cd605d8df5c183a4d760eea4eb"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.4/marchat-v1.3.4-linux-arm64.zip"
      sha256 "47bbc650ce8bff379f3b443c7322425d91ff59cbdcce46dfae38c0a55e792673"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.4/marchat-v1.3.4-linux-amd64.zip"
      sha256 "5759837e4641d647ac4f1600559e557b031f829e88cd9333f8d74745a6ad5c9d"
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
