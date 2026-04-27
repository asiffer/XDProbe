#!/bin/sh

set -e

PWD=$(cd "$(dirname "$0")" && pwd)
ROOT=$(dirname "$PWD")

echo "$ROOT"

ok() {
    printf "\033[32mOK\033[0m"
}

ko() {
    printf "\033[31mKO\033[0m"
}

log() {
    printf "\n%-50s " "$1"
}

service_exists() {
    files=$(systemctl list-unit-files xdprobe.service | grep "unit files listed" | awk '{print $1}')
    [ "$files" != "0" ]
}

from_local() {
    # copy the binary
    log "Copying xdprobe to /usr/local/bin/xdprobe"
    (sudo cp -u "$(find "${ROOT}" -name xdprobe -type f -print -quit)" /usr/local/bin/xdprobe && ok) || ko
    sudo chmod +x /usr/local/bin/xdprobe

    # copy the systemd service file
    log "Copying xdprobe systemd service file to /etc/systemd/system/xdprobe.service"
    (sudo cp -u "$(find "${PWD}" -name xdprobe.service -type f -print -quit)" /etc/systemd/system/xdprobe.service && ok) || ko
}

from_remote() {
    # download the binary
    log "Installing xdprobe to /usr/local/bin/xdprobe"
    (sudo curl -sL https://github.com/asiffer/xdprobe/releases/latest/download/xdprobe -o /usr/local/bin/xdprobe && ok) || ko
    sudo chmod +x /usr/local/bin/xdprobe

    # download the systemd service file
    log "Installing xdprobe systemd service file to /etc/systemd/system/xdprobe.service"
    (sudo curl -sL https://raw.githubusercontent.com/asiffer/xdprobe/master/systemd/xdprobe.service -o /etc/systemd/system/xdprobe.service && ok) || ko
}

install() {
    # stop the service if running so we can overwrite the binary
    log "Stopping xdprobe service if it's running"
    (sudo systemctl stop xdprobe 2>/dev/null || true) && ok

    if [ -n "${XDPROBE_LOCAL}" ]; then
        from_local
    else
        from_remote
    fi

    # download geoip database
    log "Downloading GeoIP database to /var/lib/xdprobe/geoip.mmdb"
    sudo mkdir -p /var/lib/xdprobe
    (curl -sL https://download.db-ip.com/free/dbip-city-lite-2026-04.mmdb.gz | gzip -d | sudo tee /var/lib/xdprobe/geoip.mmdb > /dev/null && ok) || ko

    # generate config
    log "Generating xdprobe configuration file at /etc/sysconfig/xdprobe"
    IFACE=$(ip -o route get 8.8.8.8 | awk '{print $5}')
    : "${PASSWORD:=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 32)}"
    HASHED_PASSWORD=$(printf "%s" "$PASSWORD" | sha256sum | awk '{print $1}')
    sudo mkdir -p /etc/sysconfig
    (printf "XDPROBE_PASSWORD=%s\nXDPROBE_GEOIP=%s\nXDPROBE_ADDR=%s\nXDPROBE_NIC=%s\n" "$HASHED_PASSWORD" "/var/lib/xdprobe/geoip.mmdb" "/run/xdprobe/xdprobe.sock" "$IFACE" | sudo tee /etc/sysconfig/xdprobe > /dev/null && ok) || ko
    printf "\n\033[1mGenerated password: %s\033[0m" "$PASSWORD" 

    # create dedicated user (if it doesn't exist) and set permissions
    log "Creating dedicated user 'xdprobe' and setting permissions"
    (id xdprobe 2>/dev/null || sudo useradd --system --no-create-home --shell /usr/sbin/nologin xdprobe && ok) || ko
    sudo chown xdprobe:xdprobe /usr/local/bin/xdprobe
    sudo chown -R xdprobe:xdprobe /var/lib/xdprobe
    sudo chown xdprobe:xdprobe /etc/sysconfig/xdprobe
    sudo chmod 600 /etc/sysconfig/xdprobe
    
    # enable and run the service
    log "Enabling and starting xdprobe systemd service"
    (sudo systemctl daemon-reload && sudo systemctl enable --quiet xdprobe && sudo systemctl start --quiet xdprobe && ok && printf "\n") || (ko && printf "\n")
}

uninstall() {
    log "Uninstalling xdprobe service"
    ( ! service_exists || (sudo systemctl stop xdprobe && sudo systemctl disable --quiet xdprobe && sudo rm -f /etc/systemd/system/xdprobe.service) && ok) || ko

    log "Removing xdprobe binary and data files"
    (sudo rm -rf /usr/local/bin/xdprobe /etc/sysconfig/xdprobe /var/lib/xdprobe /run/xdprobe && ok) || ko

    log "Removing dedicated user 'xdprobe'"
    ( ( ! id xdprobe 2>/dev/null || sudo userdel xdprobe ) && ok && printf "\n") || (ko && printf "\n") 
}

case "$1" in
    uninstall)
        uninstall
        ;;
    *)
        install
        ;;
esac

