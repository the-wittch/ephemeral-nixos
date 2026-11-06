# Contributing

Bug reports and pull requests are welcome. This is a personal project, so replies can take a few days.

- **Install problems:** use the issue form, with the end of `/tmp/ephemeral-install.log`.
- **Problems after the install:** those belong to [ephemeral-nixos-config](https://github.com/the-wittch/ephemeral-nixos-config/issues/new/choose).
- **Security problems:** see [SECURITY.md](SECURITY.md), not an issue.
- **Changes:** open an issue first for anything bigger than a fix.

## Pull requests

- This repo only holds the installer, the ISO and the install test. Changes to the installed system go to the config repo.
- Add a line to `CHANGELOG.md` under the next version.
- Run the checks in [Working on the installer](docs/development.md). For changes to the scripts, also run the install test: `gh workflow run install-test.yml --ref <branch>`, or locally.
- Keep `shellcheck` clean, and validate every new answer in `scripts/common.sh` before it reaches `settings.json` or a shell.

By contributing you agree that your work is released under the [MIT License](LICENSE).
