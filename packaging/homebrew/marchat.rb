class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/Cod-e-Codes/marchat"
  version "1.3.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.2/marchat-v1.3.2-darwin-arm64.zip"
      sha256 "309e5920024099fb141ee93b869cd891255a3a2ffb397901a404b9c376176356"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.2/marchat-v1.3.2-darwin-amd64.zip"
      sha256 "1af5c3e01feee8e6ab157370b87b09191a7b5bdbc11509cfeaccc0e4336fd21e"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.2/marchat-v1.3.2-linux-arm64.zip"
      sha256 "8893e13368016d727fed37b9e2b8fff6c75c5598aabb4b059eb1f36041945cd7"
    end
    on_intel do
      url "https://github.com/Cod-e-Codes/marchat/releases/download/v1.3.2/marchat-v1.3.2-linux-amd64.zip"
      sha256 "3f604784d11452ad907775676cb0774de8f932987e5cd208505c9d46eb89367a"
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
