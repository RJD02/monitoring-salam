#!/bin/bash
# Quick Installation Script for Target Machine

set -e

INSTALL_DIR="/opt/salam-monitoring"
SERVICE_USER="monitoring"

echo "=== Salam Unified Monitoring Platform - Quick Install ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root: sudo $0"
    exit 1
fi

echo "1. Creating installation directory..."
mkdir -p $INSTALL_DIR/{bin,config,logs,data}

echo "2. Creating service user..."
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd -r -s /bin/false -d $INSTALL_DIR $SERVICE_USER
fi

echo "3. Copying files..."
cp bin/salam-monitor $INSTALL_DIR/bin/
cp config/config.yaml $INSTALL_DIR/config/
chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR
chmod +x $INSTALL_DIR/bin/salam-monitor

echo "4. Setting up systemd service..."
cp systemd/salam-monitor.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable salam-monitor

echo "5. Creating log directory..."
mkdir -p /home/$SERVICE_USER/nfs_backup/monitoring/monitoring_util
chown -R $SERVICE_USER:$SERVICE_USER /home/$SERVICE_USER

echo ""
echo "=== Installation Complete! ==="
echo ""
echo "Next Steps:"
echo "1. Edit configuration: nano $INSTALL_DIR/config/config.yaml"
echo "   - Set mode: prod"
echo "   - Update nfs_root_prod path"
echo "   - Configure MySQL database details"
echo "   - Set Yarn ResourceManager URL"
echo ""
echo "2. Start the service: systemctl start salam-monitor"
echo "3. Check status: systemctl status salam-monitor"
echo "4. View logs: journalctl -u salam-monitor -f"
echo "5. Access web UI: http://localhost:8080"
echo ""
echo "Application logs: /home/$SERVICE_USER/nfs_backup/monitoring/monitoring_util/<date>/info.log"