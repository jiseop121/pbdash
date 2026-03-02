class PocketbaseMultiview < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "https://github.com/jiseop121/multi-pocketbase-ui"
  version "0.2.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.2.2/pbviewer-v0.2.2-darwin-arm64.tar.gz"
      sha256 "a0069e3228a3d007a5476c27f9f0f5cb97149ac9c1a2bfec749aade0d256eda4"
    else
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.2.2/pbviewer-v0.2.2-darwin-amd64.tar.gz"
      sha256 "fc7ce006aac66d6a4a6f526e300f79a2967e013f02ca4945370ac8e39bfcffaf"
    end
  end

  def install
    bin.install "pbviewer"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbviewer -c \"version\"")
  end
end
