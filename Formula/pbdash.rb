class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.4.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.2/pbdash-v0.4.2-darwin-arm64.tar.gz"
      sha256 "1b44047af90d2f77bc9fad430eb54257cc5f3843db6fd8d19b3741cda87943af"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.2/pbdash-v0.4.2-darwin-amd64.tar.gz"
      sha256 "310f3554da9537ed649f15a5f0190cea5a0e76a99c7d10fd7eee8946987d3196"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
