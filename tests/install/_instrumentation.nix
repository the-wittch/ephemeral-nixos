{ lib, modulesPath, ... }:
{
  imports = [ (modulesPath + "/testing/test-instrumentation.nix") ];
  boot.consoleLogLevel = lib.mkForce 7;
  # the passphrase prompt has to reach the serial console
  boot.plymouth.enable = lib.mkForce false;
}
