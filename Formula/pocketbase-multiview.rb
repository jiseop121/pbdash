class PocketbaseMultiview < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "https://github.com/jiseop121/multi-pocketbase-ui"
  version "0.3.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.3.0/pbviewer-v0.3.0-darwin-arm64.tar.gz"
      sha256 "08e649527cffc0490745f272f5c04d50caf6fa19251d481cda9a806cbae1157c"
    else
      url "https://github.com/jiseop121/multi-pocketbase-ui/releases/download/v0.3.0/pbviewer-v0.3.0-darwin-amd64.tar.gz"
      sha256 "299646a1f34762f6c138086d75650723526bc35f0e8ca1304eb352266ce4c373"
    end
  end

  def install
    bin.install "pbviewer"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pbviewer -c \"version\"")
  end
end
