class Enos < Formula
  desc "A tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable, modular, and declarative language."
  homepage "https://github.com/hashicorp/enos"
  version "0.0.27"

  depends_on "hashicorp/tap/terraform"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/hashicorp/enos/releases/download/v0.0.27/enos_0.0.27_darwin_arm64.zip"
    sha256 "e10ffb026f933eef8d40b25dc864557b473da344346c5c6fd54ae28062dc4932"
  end

  if OS.mac? && Hardware::CPU.intel?
    url "https://github.com/hashicorp/enos/releases/download/v0.0.27/enos_0.0.27_darwin_amd64.zip"
    sha256 "155dc543097509fb7657867c468cc0a32ea62553eadfec34326f4633a577a043"
  end

  if OS.linux? && Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
    url "https://github.com/hashicorp/enos/releases/download/v0.0.27/enos_0.0.27_linux_arm64.zip"
    sha256 "81ed9db7983b9dc5e78cb40607afb8227a002b714113c0768bcbbcfcf339f941"
  end

  if OS.linux? && Hardware::CPU.intel?
    url "https://github.com/hashicorp/enos/releases/download/v0.0.27/enos_0.0.27_linux_amd64.zip"
    sha256 "c4d4ae0d4de8315d18081a6719a634bbe263753e1f9d4a576c0b4650c2137af8"
  end

  def install
    bin.install "enos"

    generate_completions_from_executable(bin/"enos", "completion")
  end

  test do
    system "#{bin}/enos --version"
  end
end
