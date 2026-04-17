class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.0.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-darwin-arm64.zip"
      sha256 "88c0189a1e29c53e9dba03cd2b887a65f0c1e008278778c7dd12126c3807fd0a"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-darwin-amd64.zip"
      sha256 "15baf6c7c2ddaa0f939fb48e350daa7f1e7efe6d74bde8cf825b3339a4122739"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-linux-arm64.zip"
      sha256 "496d2985df4f1144c34ffeb8002f679ea05f1c66fd27119966038a468eca016c"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.0.0/marchat-v1.0.0-linux-amd64.zip"
      sha256 "bc5f1ef7fdfa50d04a9925ebab073ba2a0e1db2255f69c74f097c185f8252259"
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
