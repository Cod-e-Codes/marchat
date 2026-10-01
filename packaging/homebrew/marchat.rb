class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.8"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.8/marchat-v1.3.8-darwin-arm64.zip"
      sha256 "f50fbaf9e72286865cf25e2459cfafc5bfcf84c3f47e9f808f3443e885410381"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.8/marchat-v1.3.8-darwin-amd64.zip"
      sha256 "6b05415b36266c84bbb5d5939a3ff5ea7d3a9dd82a1a9f39c6dc6499950e5766"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.8/marchat-v1.3.8-linux-arm64.zip"
      sha256 "58cceeb97bf609c8a3b08c54032fb9d09baf87e3ad872a99b0e472c0502af7f7"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.8/marchat-v1.3.8-linux-amd64.zip"
      sha256 "47c6f14b650880115ea090a25a36fd1b746ff227b85d1f2f834fc459638f60b1"
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
