{
  lib,
  buildGoModule,
  fetchFromGitHub,
  protobuf,
  protoc-gen-go,
}:

buildGoModule rec {
  pname = "rtmp-auth";
  version = "1.3.0";
  src = ./.;

  vendorHash = "sha256-abIDkZRN3yy0l0BuZ8Y3PcKQbjqnagYcpgXU3QFaONs=";

  subPackages = [
    "cmd/rtmp-auth"
    "cmd/auth-consul-migrate"
  ];

  nativeBuildInputs = [
    protobuf
    protoc-gen-go
  ];

  preBuild = ''
    protoc -I=storage/ --go_opt=paths=source_relative --go_out=storage/ storage/storage.proto
  '';

  meta = {
    description = "Stream api for cdn automation";
    homepage = "https://github.com/voc/rtmp-auth";
    platforms = lib.platforms.linux;
    license = lib.licenses.mit;
    maintainers = with lib.maintainers; [
      ischluff
    ];
    mainProgram = "rtmp-auth";
  };
}
