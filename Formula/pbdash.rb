class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.5.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.5.1/pbdash-v0.5.1-darwin-arm64.tar.gz"
      sha256 "ae0f0dfc1d1358eb68c69141ac226226b2571e3da41851e7a3bd11e53beeaffc"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.5.1/pbdash-v0.5.1-darwin-amd64.tar.gz"
      sha256 "77c59f1aa7d88b2298dfe9a52cbacbc510d6bf4ec80c23f98327f39a6de93c34"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
