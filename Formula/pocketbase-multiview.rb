class PocketbaseMultiview < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "https://github.com/jiseop121/multi-pocketbase-ui"
  version "0.3.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.3.1/pbviewer-v0.3.1-darwin-arm64.tar.gz"
      sha256 "7f197549b8f431f60af29d025a2457828c55cff05f4aca0ddd6071149fc69585"
    else
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.3.1/pbviewer-v0.3.1-darwin-amd64.tar.gz"
      sha256 "1ea9d10a2a043c3b8b49ddc80cc7683682dcc3d8826b718aabb7190e5c1d28da"
    end
  end

  def install
    bin.install "pbviewer"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbviewer -c \"version\"")
  end
end
