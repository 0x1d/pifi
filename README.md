# PiFi

This is a simple tool to create and manage a hotspot on a Raspberry Pi (ARM64).  
The code is based on https://github.com/NetworkManager/NetworkManager/blob/main/examples/python/dbus/wifi-hotspot.py, rewritten in Go and adds following features:

- Environment variables or intput parameters for SSID and password
- Option to remove the connection

## Build

```bash
make build
```

## Usage

```bash
# sudo ./pifi <ifname> [up|down|remove] [ssid] [password]
sudo ./pifi wlan0 up "My Hotspot" "my password"
sudo ./pifi wlan0 down
sudo ./pifi wlan0 remove
```

## Environment variables

The hotspot will use the environment variables `WIFI_SSID` and `WIFI_PASSWORD` if they are set.

```bash
export WIFI_SSID="My Hotspot"
export WIFI_PASSWORD="my password"

sudo -E ./pifi wlan0 up
```
