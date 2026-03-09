class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.4.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.1/pbdash-v0.4.1-darwin-arm64.tar.gz"
      sha256 "e3204c5a809b6b3e4bdc71a8f90fbcb382d6f83b8390f6b98bcf9dd234079031"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.1/pbdash-v0.4.1-darwin-amd64.tar.gz"
      sha256 "dc3fdd20f7c0a42b639c906a52b83fb3b781dccf458c8d52efdde64265f9490e"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
