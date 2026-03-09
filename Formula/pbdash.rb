class Pbdash < Formula
  desc "Read-only CLI viewer for PocketBase instances"
  homepage "https://github.com/jiseop121/pbdash"
  version "0.4.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.0/pbdash-v0.4.0-darwin-arm64.tar.gz"
      sha256 "901742de6ea79aebc34c05606d65e5d6a5fc827acc494d1c54a58d9472ae4a8d"
    else
      url "https://github.com/jiseop121/pbdash/releases/download/v0.4.0/pbdash-v0.4.0-darwin-amd64.tar.gz"
      sha256 "2d667e8689816fc2986a54a5ca748b452b782a5a7a4d9ff9e5e6e749964bfac8"
    end
  end

  def install
    bin.install "pbdash"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbdash -c \"version\"")
  end
end
