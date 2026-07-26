#!/bin/bash
# Undo everything start-flight.sh changed.
sudo pmset -a disablesleep 0
pkill -f "uvicorn main:app" 2>/dev/null || true
pkill -f "usb_bridge.py" 2>/dev/null || true
lms unload --all 2>/dev/null || true
lms server stop 2>/dev/null || true
# Internet Sharing teardown if the GUI toggle is unreachable:
#   sudo defaults write /Library/Preferences/SystemConfiguration/com.apple.nat NAT -dict Enabled -int 0
#   sudo launchctl stop com.apple.InternetSharing
echo "done (GPU memory cap resets on reboot; Internet Sharing off via System Settings)"
