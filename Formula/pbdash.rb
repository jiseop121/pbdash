class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.4.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.1/pbdash-v0.4.1-darwin-arm64.tar.gz"
      sha256 "91784b78b7514ef7303fc5210edf2b1d04709c7efe2c52783785d1276ed2e3f0"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.1/pbdash-v0.4.1-darwin-amd64.tar.gz"
      sha256 "f573d7df576318bd0b2eb3848009631068518586c662e6f3fb9350bd033b904d"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
