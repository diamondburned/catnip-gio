# { pkgs ? import <nixpkgs> {} }:
#
# # let gotk4-nix = pkgs.fetchFromGitHub {
# # 		owner = "diamondburned";
# # 		repo  = "gotk4-nix";
# # 		rev   = "b5bb50b31ffd7202decfdb8e84a35cbe88d42c64";
# # 		hash  = "sha256:18wxf24shsra5y5hdbxqcwaf3abhvx1xla8k0vnnkrwz0r9n4iqq";
# # 	};
# let gotk4-nix = ../gotk4-nix;
#
# in import "${gotk4-nix}/shell.nix" {
# 	base = {
# 		pname = "gotkit";
# 		version = "dev";
# 	};
#
# 	buildInputs = pkgs: with pkgs; [
# 		# staticcheck takes forever to build gotk4 twice. I'm good.
# 		(writeShellScriptBin "staticcheck" "")
#
# 		# Fyne dependencies.
# 		pkg-config xorg.libX11.dev xorg.libXcursor xorg.libXi xorg.libXinerama xorg.libXrandr
# 		xorg.libXxf86vm libglvnd libxkbcommon
# 	];
# }

{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
	buildInputs = with pkgs; [
		# Gio dependencies.
		vulkan-headers
		libxkbcommon
		wayland
		xorg.libX11
		xorg.libXcursor
		xorg.libXfixes
		libGL
		pkgconfig
	];

	nativeBuildInputs = with pkgs; [
		go
		gopls
		gotools
	];

	CGO_ENABLED = "1";
}
