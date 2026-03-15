{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    {
      devShells.x86_64-linux.default =
        let
          pkgs = nixpkgs.legacyPackages.x86_64-linux;
        in
        pkgs.mkShell {
          buildInputs = with pkgs; [
            # Gio dependencies.
            vulkan-headers
            libxkbcommon
            wayland
            xorg.libX11
            xorg.libXcursor
            xorg.libXfixes
            libxcb
            libGL
            pkg-config
          ];

          nativeBuildInputs = with pkgs; [
            go
            gopls
            gotools
          ];

          CGO_ENABLED = "1";
        };
    };
}
