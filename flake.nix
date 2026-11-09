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

        # Nix flakes always strip .git from the copied source, so the commit
        # comes from self.rev/self.dirtyRev and its timestamp from
        # self.lastModifiedDate (UTC, YYYYMMDDHHMMSS). There's no flake
        # attribute for the nearest tag, so `task nix-build` writes
        # `git describe --tags --long --dirty` (e.g. "v0.23.3-38-g818025e")
        # to an untracked VERSION file, which survives the source copy.
        # Staging that file makes nix see every tree as dirty, so when it
        # exists its "-dirty" suffix decides dirtiness instead.
        #
        # From those this builds the same version a plain `go build` stamps:
        #   at a tag:        v0.23.3
        #   after a tag:     v0.23.4-0.20261103201002-818025e9fe34
        #   no VERSION file: v0.0.0-20261103201002-818025e9fe34
        # i.e. a Go pseudo-version, which sorts after the last release and
        # before the next, so it can't be mistaken for a release. A dirty
        # tree appends "+dev".
        lib = pkgs.lib;
        hasVersionFile = builtins.pathExists ./VERSION;
        describeRaw = if hasVersionFile then lib.trim (builtins.readFile ./VERSION) else "";
        describe = lib.removeSuffix "-dirty" describeRaw;
        dirty = if hasVersionFile then lib.hasSuffix "-dirty" describeRaw else self ? dirtyRev;
        rev = self.rev or (lib.removeSuffix "-dirty" (self.dirtyRev or ""));
        gitCommit = if rev == "" then "unknown" else rev;
        gitTreeState = if dirty then "dirty" else "clean";
        # Commit time as RFC 3339, matching what a plain `go build` reports.
        d = self.lastModifiedDate;
        buildDate = "${builtins.substring 0 4 d}-${builtins.substring 4 2 d}-${builtins.substring 6 2 d}T${builtins.substring 8 2 d}:${builtins.substring 10 2 d}:${builtins.substring 12 2 d}Z";
        pseudoSuffix = "${self.lastModifiedDate}-${builtins.substring 0 12 rev}";

        # [ tag commitsSinceTag ], or null for an empty or unrecognized VERSION.
        described = builtins.match "(v[0-9]+\\.[0-9]+\\.[0-9]+.*)-([0-9]+)-g[0-9a-f]+" describe;
        # [ major minor patch prerelease-or-null ]
        tagParts = builtins.match "v([0-9]+)\\.([0-9]+)\\.([0-9]+)(-.*)?" (builtins.elemAt described 0);

        pseudoVersion =
          if described == null || tagParts == null then "v0.0.0-${pseudoSuffix}"
          else if builtins.elemAt described 1 == "0" then builtins.elemAt described 0
          else if builtins.elemAt tagParts 3 != null then "${builtins.elemAt described 0}.0.${pseudoSuffix}"
          else
            let
              patch = lib.toInt (builtins.elemAt tagParts 2) + 1;
            in
            "v${builtins.elemAt tagParts 0}.${builtins.elemAt tagParts 1}.${toString patch}-0.${pseudoSuffix}";

        gitVersion =
          if rev == "" then "v0.0.0"
          else pseudoVersion + lib.optionalString dirty "+dev";

      in
      {
        packages = {
          default = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
            pname = "datumctl";
            version = gitVersion;

            src = ./.;

            # Hash of Go module dependencies.
            # Update this after changing go.mod/go.sum:
            #   task nix-update-hash
            vendorHash = "sha256-YPQxXR1V4zbvUZoE2NL1LDHqRffEqYEoTK5iB7qwXVk=";

            env.CGO_ENABLED = 0;

            tags = [ "netgo" "osusergo" ];

            ldflags = [
              "-s"
              "-w"
              "-X k8s.io/component-base/version.gitVersion=${gitVersion}"
              "-X k8s.io/component-base/version.gitCommit=${gitCommit}"
              "-X k8s.io/component-base/version.gitTreeState=${gitTreeState}"
              "-X k8s.io/component-base/version.buildDate=${buildDate}"
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
            go_1_27
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
