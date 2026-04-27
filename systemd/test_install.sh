#!/bin/sh

PASS=0
FAIL=0

ok() {
    printf "\033[32mOK\033[0m\n"
    PASS=$((PASS + 1))
}

ko() {
    printf "\033[31mKO\033[0m %s\n" "$1"
    FAIL=$((FAIL + 1))
}

check() {
    printf "%-60s " "$1"
}

# --- binary ---

check "Binary exists at /usr/local/bin/xdprobe"
if [ -f /usr/local/bin/xdprobe ]; then ok; else ko "file not found"; fi

check "Binary is executable"
if [ -x /usr/local/bin/xdprobe ]; then ok; else ko "not executable"; fi

check "Binary owner is xdprobe:xdprobe"
owner=$(stat -c '%U:%G' /usr/local/bin/xdprobe 2>/dev/null || echo "unknown")
if [ "$owner" = "xdprobe:xdprobe" ]; then ok; else ko "owner is '$owner'"; fi

# --- systemd service ---

check "Service file exists at /etc/systemd/system/xdprobe.service"
if [ -f /etc/systemd/system/xdprobe.service ]; then ok; else ko "file not found"; fi

check "Service is enabled"
if systemctl is-enabled --quiet xdprobe 2>/dev/null; then ok; else ko "not enabled"; fi

check "Service is running"
if systemctl is-active --quiet xdprobe 2>/dev/null; then ok; else ko "not active"; fi

# --- geoip database ---

check "GeoIP database exists at /var/lib/xdprobe/geoip.mmdb"
if [ -f /var/lib/xdprobe/geoip.mmdb ]; then ok; else ko "file not found"; fi

check "GeoIP database is non-empty"
if [ -s /var/lib/xdprobe/geoip.mmdb ]; then ok; else ko "file is empty"; fi

check "/var/lib/xdprobe owner is xdprobe:xdprobe"
dir_owner=$(stat -c '%U:%G' /var/lib/xdprobe 2>/dev/null || echo "unknown")
if [ "$dir_owner" = "xdprobe:xdprobe" ]; then ok; else ko "owner is '$dir_owner'"; fi

# --- config ---

check "Config file exists at /etc/sysconfig/xdprobe"
if [ -f /etc/sysconfig/xdprobe ]; then ok; else ko "file not found"; fi

check "Config file permissions are 600"
perms=$(stat -c '%a' /etc/sysconfig/xdprobe 2>/dev/null || echo "unknown")
if [ "$perms" = "600" ]; then ok; else ko "permissions are '$perms'"; fi

check "Config file owner is xdprobe:xdprobe"
cfg_owner=$(stat -c '%U:%G' /etc/sysconfig/xdprobe 2>/dev/null || echo "unknown")
if [ "$cfg_owner" = "xdprobe:xdprobe" ]; then ok; else ko "owner is '$cfg_owner'"; fi

check "Config contains XDPROBE_PASSWORD"
if sudo grep -q "^XDPROBE_PASSWORD=" /etc/sysconfig/xdprobe 2>/dev/null; then ok; else ko "key missing"; fi

check "Config contains XDPROBE_GEOIP"
if sudo grep -q "^XDPROBE_GEOIP=" /etc/sysconfig/xdprobe 2>/dev/null; then ok; else ko "key missing"; fi

check "Config contains XDPROBE_ADDR"
if sudo grep -q "^XDPROBE_ADDR=" /etc/sysconfig/xdprobe 2>/dev/null; then ok; else ko "key missing"; fi

check "Config contains XDPROBE_NIC"
if sudo grep -q "^XDPROBE_NIC=" /etc/sysconfig/xdprobe 2>/dev/null; then ok; else ko "key missing"; fi

# --- user ---

check "System user 'xdprobe' exists"
if id xdprobe >/dev/null 2>&1; then ok; else ko "user not found"; fi

check "User 'xdprobe' has no login shell"
shell=$(getent passwd xdprobe | cut -d: -f7)
if [ "$shell" = "/usr/sbin/nologin" ] || [ "$shell" = "/sbin/nologin" ]; then ok; else ko "shell is '$shell'"; fi

# --- summary ---

printf "\n%d passed, %d failed\n" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]