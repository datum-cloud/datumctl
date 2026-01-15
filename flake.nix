{
  description = "datumctl - A CLI for interacting with Datum Cloud";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ] (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        # Get version from git, fallback to "dev" if not in a git repo
        version =
          if (builtins.pathExists ./.git)
          then builtins.replaceStrings ["\n"] [""] (builtins.readFile (
            pkgs.runCommand "get-version" {} ''
              cd ${./.}
              ${pkgs.git}/bin/git describe --tags --always --dirty 2>/dev/null > $out || echo "dev" > $out
            ''
          ))
          else "dev";

      in
      {
        packages = {
          default = pkgs.buildGoModule {
            pname = "datumctl";
            inherit version;

            src = ./.;

            # Hash of Go module dependencies.
            # Update this after changing go.mod/go.sum:
            #   go run bin/update-nix-hash.go
            vendorHash = "sha256-IZtck6ZsaIoEZLpukWHVbQAhfOsly0WO0OWO+6uRhgE=";

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            meta = with pkgs.lib; {
              description = "A CLI for interacting with the Datum platform";
              homepage = "https://www.datum.net/docs/quickstart/datumctl/";
              license = licenses.asl20;
              maintainers = [ ];
              mainProgram = "datumctl";
            };
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_25
            gopls
            gotools
            go-tools
            git
          ];

          shellHook = ''
            echo "datumctl development environment"
            echo "Go version: $(go version)"
          '';
        };

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
