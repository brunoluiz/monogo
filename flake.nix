{
  description = "Example Go development environment for Zero to Nix";

  # Flake inputs
  inputs = {
    # Latest stable Nixpkgs
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0";
  };

  # Flake outputs
  outputs = { self, nixpkgs }:
    let
      # Systems supported
      allSystems = [
        "x86_64-linux" # 64-bit Intel/AMD Linux
        "aarch64-linux" # 64-bit ARM Linux
        "aarch64-darwin" # 64-bit ARM macOS
      ];

      # Helper to provide system-specific attributes
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      # Development environment output
      devShells = forAllSystems ({ pkgs }: {
        default = pkgs.mkShell {
          shellHook = ''
          unset GOROOT
          '';
          # The Nix packages provided in the environment
          packages = with pkgs; [
            go
            gotools
            gitMinimal
            golangci-lint
          ];
        };
      });

      packages = forAllSystems ({ pkgs }: {
        default = pkgs.buildGoModule {
          name = "monogo";
          src = self;
          goSum = ./go.sum;
          vendorHash = "sha256-DEo2Y8RbwUa4BVLwPYUbkFQ2bYug6LrzWsyfC4PUPGI=";
           subPackages = [ "cmd/monogo" ];
        };
      });
    };
}
