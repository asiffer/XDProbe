#!/bin/sh

set -e

ok() {
    printf "\033[32mOK\033[0m"
}

ko() {
    printf "\033[31mKO\033[0m"
}

log() {
    printf "\n%-50s " "$1"
}

install() {
    # download the binary 
    log "Installing xdprobe to /usr/local/bin/xdprobe"
    (sudo curl -sL https://github.com/asiffer/xdprobe/releases/latest/download/xdprobe -o /usr/local/bin/xdprobe && ok) || ko
    sudo chmod +x /usr/local/bin/xdprobe

    # download the systemd service file
    log "Installing xdprobe systemd service file to /etc/systemd/system/xdprobe.service"
    (sudo curl -sL https://raw.githubusercontent.com/asiffer/xdprobe/master/systemd/xdprobe.service -o /etc/systemd/system/xdprobe.service && ok) || ko

    # download geoip database
    log "Downloading GeoIP database to /var/lib/xdprobe/geoip.mmdb"
    sudo mkdir -p /var/lib/xdprobe
    (curl -sL https://download.db-ip.com/free/dbip-city-lite-2026-04.mmdb.gz | gzip -d | sudo tee /var/lib/xdprobe/geoip.mmdb > /dev/null && ok) || ko

    # prepare socket directory
    sudo mkdir -p /run/xdprobe

    # generate config
    log "Generating xdprobe configuration file at /etc/sysconfig/xdprobe"
    STRONG_PASSWORD=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 32)
    (printf "XDPROBE_PASSWORD=%s\nXDPROBE_GEOIP_DB=%s\nXDPROBE_ADDR=%s\n" "$STRONG_PASSWORD" "/var/lib/xdprobe/geoip.mmdb" "/run/xdprobe.sock" | sudo tee /etc/sysconfig/xdprobe > /dev/null && ok) || ko
    printf "\n\033[1mGenerated password: %s\033[0m" "$STRONG_PASSWORD"

    # create dedicated user (if it doesn't exist) and set permissions
    log "Creating dedicated user 'xdprobe' and setting permissions"
    (id xdprobe 2>/dev/null || sudo useradd --system --no-create-home --shell /usr/sbin/nologin xdprobe && ok) || ko
    sudo chown xdprobe:xdprobe /usr/local/bin/xdprobe
    sudo chown -R xdprobe:xdprobe /var/lib/xdprobe
    sudo chown xdprobe:xdprobe /run/xdprobe
    sudo chown xdprobe:xdprobe /etc/sysconfig/xdprobe
    sudo chmod 600 /etc/sysconfig/xdprobe
    

    # enable and run the service
    log "Enabling and starting xdprobe systemd service"
    (sudo systemctl daemon-reload && sudo systemctl enable xdprobe && sudo systemctl start xdprobe && ok && printf "\n") || (ko && printf "\n")
}

uninstall() {
    log "Uninstalling xdprobe service"
    (sudo systemctl stop xdprobe && sudo systemctl disable xdprobe && sudo rm -f /etc/systemd/system/xdprobe.service && ok) || ko

    log "Removing xdprobe binary and data files"
    (sudo rm -rf /usr/local/bin/xdprobe /etc/sysconfig/xdprobe /var/lib/xdprobe /run/xdprobe && ok) || ko

    log "Removing dedicated user 'xdprobe'"
    (sudo userdel xdprobe && ok && printf "\n") || (ko && printf "\n") 
}

case "$1" in
    uninstall)
        uninstall
        ;;
    *)
        install
        ;;
esac

