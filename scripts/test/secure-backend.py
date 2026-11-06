#!/usr/bin/env python3
"""Exercise synthetic captures on a disposable Linux Secret Service/session bus.

Requires dbus-run-session, gnome-keyring-daemon, secret-tool, gdbus and Go.
Never connects to the operator's existing credential session or uses real secrets.
"""

import os
import pathlib
import re
import signal
import subprocess
import sys
import tempfile


def run():
    if "--inside" not in sys.argv:
        temporary_root = "/tmp/opencode" if pathlib.Path("/tmp/opencode").exists() else None
        with tempfile.TemporaryDirectory(prefix="burrow-keyring-", dir=temporary_root) as directory:
            env = os.environ.copy()
            env["GOMODCACHE"] = subprocess.check_output(["go", "env", "GOMODCACHE"], text=True).strip()
            env["GOCACHE"] = subprocess.check_output(["go", "env", "GOCACHE"], text=True).strip()
            env.update(HOME=directory, XDG_RUNTIME_DIR=directory + "/runtime", XDG_DATA_HOME=directory + "/data", XDG_CONFIG_HOME=directory + "/config", GNOME_KEYRING_CONTROL=directory + "/runtime/keyring")
            for key in ("DBUS_SESSION_BUS_ADDRESS", "DBUS_STARTER_ADDRESS", "DBUS_STARTER_BUS_TYPE"):
                env.pop(key, None)
            pathlib.Path(env["XDG_RUNTIME_DIR"]).mkdir(mode=0o700)
            return subprocess.run(["dbus-run-session", "--", sys.executable, __file__, "--inside"], env=env).returncode

    control = os.environ["GNOME_KEYRING_CONTROL"]
    daemon = subprocess.Popen(["gnome-keyring-daemon", "--foreground", "--unlock", "--components=secrets", "--control-directory=" + control], stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    service_pid = None
    try:
        daemon.stdin.write(b"synthetic-test-keyring-password\n")
        daemon.stdin.close()
        env = os.environ.copy()
        env.update(BURROW_TEST_SECURE_BACKEND="1", GOTOOLCHAIN="go1.26.2")
        probe = subprocess.run(["secret-tool", "store", "--label=Burrow acceptance probe", "burrow-test", "probe"], input=b"synthetic-probe", env=env, timeout=15, capture_output=True)
        if probe.returncode:
            print("Isolated credential backend probe failed", file=sys.stderr)
            return 1
        owner = subprocess.check_output(["gdbus", "call", "--session", "--dest", "org.freedesktop.DBus", "--object-path", "/org/freedesktop/DBus", "--method", "org.freedesktop.DBus.GetConnectionUnixProcessID", "org.freedesktop.secrets"], text=True, env=env, timeout=5)
        match = re.fullmatch(r"\(uint32 (\d+),\)\s*", owner)
        if not match:
            raise RuntimeError("Cannot identify disposable credential service")
        service_pid = int(match.group(1))
        tests = [("./burrowctl/cmd", "TestWorkflowEncryptedCaptureAndDurableReplayWithSecureBackend"), ("./burrowctl/internal/capture", "TestOSKeysLockedBackendRefusesWithoutPrompt")]
        if env.get("BURROW_TEST_REDIS"):
            tests.insert(1, ("./burrowd/internal", "TestCombinedWorkflowsRealServices"))
        if env.get("BURROW_TEST_LIVENESS") == "1":
            tests.insert(-1, ("./burrowd/internal", "TestPersistentPartitionExpiresWithinTransportBound"))
        tests.insert(-1, ("./burrowd/internal", "TestNativeCaptureTUIUsesDurableReplayAndStableSelection"))
        for package, test in tests:
            result = subprocess.run(["go", "test", package, "-run", test, "-count=1", "-v", "-timeout=45s"], env=env, timeout=120)
            if result.returncode:
                return result.returncode
        return 0
    finally:
        # This PID was resolved only on the newly-created private bus above.
        if service_pid is not None and service_pid != daemon.pid:
            try:
                os.kill(service_pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
        daemon.terminate()
        try:
            daemon.wait(timeout=5)
        except subprocess.TimeoutExpired:
            daemon.kill()
            daemon.wait()


if __name__ == "__main__":
    raise SystemExit(run())
