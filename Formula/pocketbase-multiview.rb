class PocketbaseMultiview < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "https://github.com/jiseop121/multi-pocketbase-ui"
  version "0.3.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.3.0/pbviewer-v0.3.0-darwin-arm64.tar.gz"
      sha256 "491800d0ee45ca18c61ba76f30e3b52a006d604344afb8ddf537ce73b2ab83d7"
    else
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.3.0/pbviewer-v0.3.0-darwin-amd64.tar.gz"
      sha256 "da686b24bba9271b427904616a41d20d594b1e21d7a5d414a5a9861c434ffa01"
    end
  end

  def install
    bin.install "pbviewer"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbviewer -c \"version\"")
  end
end
