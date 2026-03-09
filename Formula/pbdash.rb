class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.4.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.1/pbdash-v0.4.1-darwin-arm64.tar.gz"
      sha256 "718981360eb8b5c4a00ba6193a33f78109aaec34a86d2a408921c0404ee85b6f"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.1/pbdash-v0.4.1-darwin-amd64.tar.gz"
      sha256 "7aa34ec79cacf87798f0139ab00b5eed9e6e0f4182243f07605bb9a79f688ca6"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
