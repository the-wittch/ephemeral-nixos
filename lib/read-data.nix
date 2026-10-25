# machine files are json, edited with jq. a .nix of the same name still
# loads, with a warning.
{ lib }:

base: default:
let
  name = baseNameOf (toString base);
  json = base + ".json";
  legacy = base + ".nix";
in
if builtins.pathExists json then
  builtins.fromJSON (builtins.readFile json)
else if builtins.pathExists legacy then
  lib.warn (
    "${name}.nix is deprecated. Convert it once, then delete the .nix file:\n"
    + "  nix eval --json --file ${name}.nix | nix run nixpkgs#jq -- . > ${name}.json"
  ) (import legacy)
else
  default
