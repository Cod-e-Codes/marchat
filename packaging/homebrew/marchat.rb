class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.0/marchat-v1.3.0-darwin-arm64.zip"
      sha256 "f2b3477e36ef13dd0ce4a1115f18d7bc4dc12df2b1b946709264fba46eb45884"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.0/marchat-v1.3.0-darwin-amd64.zip"
      sha256 "7b117e7b52a26b699c04865a2ce333b7a4e610e5ffd84b9c945be0d58a8df89f"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.0/marchat-v1.3.0-linux-arm64.zip"
      sha256 "dbe80c2a6a54aef7732274d4af39ebb6740402aef6b797a7e19ad8ffd14b33c4"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.0/marchat-v1.3.0-linux-amd64.zip"
      sha256 "6375cfa4045d0f70ea1a2d59545cbb4c7b17fd1836d0ae7cf5411ac5243afe61"
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
