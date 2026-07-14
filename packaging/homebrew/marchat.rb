class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.1/marchat-v1.3.1-darwin-arm64.zip"
      sha256 "776ae75313e633d74924081299e621b4a25d2d34593bb6258954c5f8925f5cd5"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.1/marchat-v1.3.1-darwin-amd64.zip"
      sha256 "07e0151f6c8d17430a864fd2629f1b584a669a843638d88a9ed14c4ee0a75fb3"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.1/marchat-v1.3.1-linux-arm64.zip"
      sha256 "cc882210c5849f1d8aaedbc51440597b301cf332db220ad067f7b2c102e90d8c"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.1/marchat-v1.3.1-linux-amd64.zip"
      sha256 "c5d7d1b85c39eb4d9569624a9e1546281734df5c22f4ce2fe91768d0bca8bf85"
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
