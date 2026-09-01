class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.6"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.6/marchat-v1.3.6-darwin-arm64.zip"
      sha256 "97b236852decd4e05621ca24c4a910b6c5047516976b015f5c5ab3475d6f7c35"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.6/marchat-v1.3.6-darwin-amd64.zip"
      sha256 "d5fcca4324fc6863e5ac19b95a6ed0d67124c5b6807412608ab01fffaae0c504"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.6/marchat-v1.3.6-linux-arm64.zip"
      sha256 "5b8a2ab693ade484704ddf8c45d47272928a4a551b341d122d71dbf7a2a5d7f3"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.6/marchat-v1.3.6-linux-amd64.zip"
      sha256 "a7e014e8ef161cea306c2cd2b08ab1a0265c852d10db0f1753f33963dde18ddf"
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
