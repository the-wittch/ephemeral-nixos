{ ... }:

{
  # no kexec or /dev/mem writes. also blocks hibernation
  security.protectKernelImage = true;

  boot.kernel.sysctl = {
    # root only
    "kernel.kptr_restrict" = 2;
    "kernel.dmesg_restrict" = 1;
    # no unprivileged bpf
    "kernel.unprivileged_bpf_disabled" = 1;
    "net.core.bpf_jit_harden" = 2;
    # ptrace own children only
    "kernel.yama.ptrace_scope" = 1;
    # no core dumps from setuid programs
    "fs.suid_dumpable" = 0;
    # stricter than the systemd defaults
    "fs.protected_fifos" = 2;
    "fs.protected_regular" = 2;
    # no tty line discipline autoload
    "dev.tty.ldisc_autoload" = 0;
    # not a router
    "net.ipv4.conf.all.accept_redirects" = 0;
    "net.ipv4.conf.default.accept_redirects" = 0;
    "net.ipv4.conf.all.secure_redirects" = 0;
    "net.ipv4.conf.default.secure_redirects" = 0;
    "net.ipv6.conf.all.accept_redirects" = 0;
    "net.ipv6.conf.default.accept_redirects" = 0;
    "net.ipv4.conf.all.send_redirects" = 0;
    "net.ipv4.conf.default.send_redirects" = 0;
    "net.ipv4.conf.all.accept_source_route" = 0;
    "net.ipv6.conf.all.accept_source_route" = 0;
  };

  # rare protocols with a history of bugs. modprobe still works
  boot.blacklistedKernelModules = [
    "dccp"
    "sctp"
    "rds"
    "tipc"
  ];
}
