class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.6.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.5.2/pbdash-v0.5.2-darwin-arm64.tar.gz"
      sha256 "19ed6ec26e337b9a06f77292e4b2ecc7170c6a96feba3ce9881ac4e478d950e7"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.5.2/pbdash-v0.5.2-darwin-amd64.tar.gz"
      sha256 "f71612df30601ad86c99c75a2d1cf34306fa3856b46a7bd1fc64c15d7b8c0a34"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
