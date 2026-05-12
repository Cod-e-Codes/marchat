class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.1.0/marchat-v1.1.0-darwin-arm64.zip"
      sha256 "3778163b429d4971ae0c6ecf513c56e52aeb96c6aa90778f7033b2316fd0e347"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.1.0/marchat-v1.1.0-darwin-amd64.zip"
      sha256 "c4591bb016c1f6cb6d299389af798a7f020833c0fe14e4912a11a348ff08486a"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.1.0/marchat-v1.1.0-linux-arm64.zip"
      sha256 "76efd9de78da9e5b7065969371598f172a24d971dd5baf91cdd7a36a02229b0a"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.1.0/marchat-v1.1.0-linux-amd64.zip"
      sha256 "66d8d6b08746e087d7831c3e705956f46068d35bfeea75c7154f31143be70719"
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
