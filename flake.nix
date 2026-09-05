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

        # Nix flakes always strip .git from the copied source (self/./. here has
        # no .git even though the real checkout does), so `self.rev`/`dirtyRev`
        # -- populated by the flake's git-fetcher from outside that copy -- are
        # the only reliable way to get the commit sha. There's no equivalent
        # flake attribute for "nearest git tag", so that comes from an optional
        # untracked VERSION file (`git describe --tags --abbrev=0 > VERSION`,
        # not committed) which -- being an ordinary file -- does survive the
        # source copy; absent that file this falls back to a valid placeholder.
        # Either way the result must be semver-shaped ("vX.Y.Z[+meta]"), since
        # k8s.io/component-base/version (used by `datumctl version`) parses it
        # strictly and a bare "dev"/"unknown" would fail that parse.
        # self.dirtyRev/dirtyShortRev append a literal "-dirty" suffix when the
        # working tree has uncommitted changes; swap it for "-dev" everywhere.
        markDev = builtins.replaceStrings [ "-dirty" ] [ "-dev" ];
        gitCommit = markDev (self.dirtyRev or self.rev or "unknown");
        gitSha = markDev (self.dirtyShortRev or self.shortRev or "unknown");

        lastTag =
          if (builtins.pathExists ./VERSION)
          then builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION)
          else "v0.0.0";

        gitVersion = "${lastTag}+${gitSha}";

      in
      {
        packages = {
          default = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
            pname = "datumctl";
            version = gitVersion;

            src = ./.;

            # Hash of Go module dependencies.
            # Update this after changing go.mod/go.sum:
            #   task nix-update-hash
            vendorHash = "sha256-COLwu6w7Vwqa35uCjam1d1oyGqGTtM5JqD3N1mgRusM=";

            env.CGO_ENABLED = 0;

            tags = [ "netgo" "osusergo" ];

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${gitVersion}"
              "-X k8s.io/component-base/version.gitVersion=${gitVersion}"
              "-X k8s.io/component-base/version.gitCommit=${gitCommit}"
              "-extldflags=-static"
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
            go_1_26
            gopls
            gotools
            go-task
            go-tools
            goreleaser
            syft
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
