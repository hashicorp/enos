require_relative "../Strategies/private_strategy"
class Enos < Formula
	desc "A tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable and shareable declarative language."
	homepage "https://github.com/hashicorp/enos"
	version "0.0.1"

	on_macos do
		if Hardware::CPU.arm?
			url "https://github.com/hashicorp/enos/releases/download/v0.0.1/enos_0.0.1_darwin_arm64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "15d82aa03f5585966bc747e04ee31c025391dc1c80b3ba6419c95f6b764eebbd"

			def install
				bin.install "enos"
			end
		end
		if Hardware::CPU.intel?
			url "https://github.com/hashicorp/enos/releases/download/v0.0.1/enos_0.0.1_darwin_amd64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "788eda2be1887fa13b2aca2a5bcad4535278310946c8f6f68fa561e72f7a351b"

			def install
				bin.install "enos"
			end
		end
	end

	on_linux do
		if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
			url "https://github.com/hashicorp/enos/releases/download/v0.0.1/enos_0.0.1_linux_arm64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "2a1677b83d6ec24038ef949420afa79269a405e35396aefec432412233cfc251"

			def install
				bin.install "enos"
			end
		end
		if Hardware::CPU.intel?
			url "https://github.com/hashicorp/enos/releases/download/v0.0.1/enos_0.0.1_linux_amd64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "dc1f9597b024b59bf444d894c633c3fe796b7d57d74d7983ad57c4d7d37a516d"

			def install
				bin.install "enos"
			end
		end
	end
end
