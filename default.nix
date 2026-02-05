{ pkgs ? import <nixpkgs> {} }:

pkgs.buildGoModule {
  pname = "context";
  version = "0.1.0";
  src = ./.;
  
  vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
  
  ldflags = [
    "-s"
    "-w"
  ];
  
  postInstall = ''
    mkdir -p $out/share/context/shell
    cp -r $src/shell/* $out/share/context/shell/
  '';
  
  meta = with pkgs.lib; {
    description = "Terminal context capture tool for AI-assisted debugging";
    homepage = "https://github.com/jupiterozeye/context";
    license = licenses.mit;
  };
}