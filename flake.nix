{
  description = "soft-serve dev environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"
      ];

      coverageScript = pkgs: pkgs.writeShellApplication {
        name = "coverage";
        runtimeInputs = [ pkgs.go ];
        text = ''
          # We collect coverage data from two sources,
          # 1) unit tests 2) integration tests
          #
          # https://go.dev/testing/coverage/
          # https://dustinspecker.com/posts/go-combined-unit-integration-code-coverage/
          # https://github.com/golang/go/issues/51430#issuecomment-1344711300
          mkdir -p coverage/unit coverage/int coverage/int2

          # Collect unit tests coverage
          go test -failfast -race -timeout 5m -skip=^TestScript -cover ./... \
            -args -test.gocoverdir="$PWD/coverage/unit"

          # Collect integration tests coverage
          GOCOVERDIR="$PWD/coverage/int" go test -failfast -race -timeout 5m -run=^TestScript ./...
          SOFT_SERVE_DB_DRIVER=postgres \
            SOFT_SERVE_DB_DATA_SOURCE="postgres://postgres:postgres@localhost/postgres?sslmode=disable" \
            GOCOVERDIR="$PWD/coverage/int2" go test -failfast -race -timeout 5m -run=^TestScript ./...

          # Convert coverage data to legacy textfmt format to upload
          go tool covdata textfmt -i=coverage/unit,coverage/int,coverage/int2 -o=coverage.txt
        '';
      };
    in {
      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              golangci-lint
              go-task
              git
              git-lfs
              gh
            ];
          };
        });

      apps = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          coverage = {
            type = "app";
            program = "${coverageScript pkgs}/bin/coverage";
          };
        });

      checks = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          coverage-script = coverageScript pkgs;
        });
    };
}
