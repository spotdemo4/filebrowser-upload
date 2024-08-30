{
  description = "Filebrowser upload";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, gomod2nix, gitignore }:
    let
      allSystems = [ 
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        inherit system;
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      # Dev environment
      devShell = forAllSystems ({ system, pkgs }:
        pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gotools
            gopls
          ];
        }
      );

      # Package
      packages = forAllSystems ({ system, pkgs, ... }:
        let
          buildGoApplication = gomod2nix.legacyPackages.${system}.buildGoApplication;
        in
        rec {
          default = filebrowser-upload;

          filebrowser-upload = buildGoApplication {
            name = "filebrowser-upload";
            src = gitignore.lib.gitignoreSource ./.;
            go = pkgs.go;
            pwd = ./.;
          };
        }
      );

      # Overlay
      overlays.default = final: prev: {
        filebrowser-upload = self.packages.${final.stdenv.system}.filebrowser-upload;
      };
    };
}
