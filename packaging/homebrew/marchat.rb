class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.5"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.5/marchat-v1.3.5-darwin-arm64.zip"
      sha256 "6fdf9a8852912f206af3870f9305dfe513ad2821c44c40e4790a6d4ab2d88441"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.5/marchat-v1.3.5-darwin-amd64.zip"
      sha256 "376639cce0653f718a0891c6c35b3b49ba41a6090ee6a64502ffef897a826146"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.5/marchat-v1.3.5-linux-arm64.zip"
      sha256 "2664c9f1af04bd651b5e6fa48df99b1793cd91190aa1f0b985a5ae4577cc0ac5"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.5/marchat-v1.3.5-linux-amd64.zip"
      sha256 "7d699faea2b50e0a009aadef6f4816a9a17b98a5f202667a45efccfeb3be5c5d"
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
