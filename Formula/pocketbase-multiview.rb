class PocketbaseMultiview < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "https://github.com/jiseop121/multi-pocketbase-ui"
  version "0.2.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.2.2/pbviewer-v0.2.2-darwin-arm64.tar.gz"
      sha256 "203493fc8cadbce16c87da9c3e1634e17d4660f83f57477dd44266d3527b1e97"
    else
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.2.2/pbviewer-v0.2.2-darwin-amd64.tar.gz"
      sha256 "28455f9aa29aa5882287f0d3695c99811b27118efd441f057a499f91242e3a9b"
    end
  end

  def install
    bin.install "pbviewer"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbviewer -c \"version\"")
  end
end
