class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.6.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.6.0/pbdash-v0.6.0-darwin-arm64.tar.gz"
      sha256 "fb0ab1689b287b23745562a0fb347f7ef0766175eec9bd848d86a488071b9f43"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.6.0/pbdash-v0.6.0-darwin-amd64.tar.gz"
      sha256 "60bb2452bcf27b1fa11fbc52ea59f9030f5cc111c0844b7db10a225ad74d655e"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
