class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.3/marchat-v1.3.3-darwin-arm64.zip"
      sha256 "9e07b783493d538309d66aae1918ad75658683622dcc153c36c43a44014aaa4b"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.3/marchat-v1.3.3-darwin-amd64.zip"
      sha256 "a350a8bcedf35bbf85b5f2229972c21d0c056de3225b4fd5812113fa743cba06"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.3/marchat-v1.3.3-linux-arm64.zip"
      sha256 "ff5f8ebd441c0d81247389fdd182431c79f25ea1a10d61333136c99e60baa302"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.3/marchat-v1.3.3-linux-amd64.zip"
      sha256 "324316296759dad4fb4380953e812be185e95ae9de3ef831ca090eca6623245d"
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
