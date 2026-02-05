{
  description = "Context - Terminal context capture tool for AI-assisted debugging";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = nixpkgs.legacyPackages.${system};
      version = "0.1.0";
    in {
      packages = {
        context = pkgs.buildGoModule {
          pname = "context";
          inherit version;
          src = ./.;

          vendorHash = "sha256-uJ2QwlRmyocxCEgoDT15giDDr6GkveTGdmdPTAZVW2w=";

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          buildInputs = with pkgs; [wl-clipboard];

          postInstall = ''
            wrapProgram $out/bin/context \
              --prefix PATH : ${pkgs.lib.makeBinPath (with pkgs; [wl-clipboard xclip])}
          '';

          nativeBuildInputs = [pkgs.makeWrapper];

          meta = with pkgs.lib; {
            description = "Terminal context capture tool for AI-assisted debugging";
            homepage = "https://github.com/jupiterozeye/context";
            license = licenses.mit;
            maintainers = [];
            platforms = platforms.all;
          };
        };
        default = self.packages.${system}.context;
      };

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          gopls
          gofumpt
          golangci-lint
        ];
      };
    });
}

