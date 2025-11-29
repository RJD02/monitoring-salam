#!/bin/bash

# Salam Monitoring Platform - Production Deployment Script
# Usage: ./deploy.sh [target_host]

set -e  # Exit on any error

# Configuration
TARGET_HOST=${1:-"production-server"}
APP_NAME="salam-monitor"
DEPLOY_DIR="/opt/salam-monitoring"
SERVICE_NAME="salam-monitor"
BACKUP_DIR="/opt/salam-monitoring-backup-$(date +%Y%m%d-%H%M%S)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if [ ! -f "$APP_NAME" ]; then
        log_error "Binary $APP_NAME not found. Please run 'go build -o $APP_NAME cmd/main.go' first"
        exit 1
    fi
    
    if [ ! -f "config/config.yaml" ]; then
        log_error "Configuration file config/config.yaml not found"
        exit 1
    fi
    
    if [ ! -f "systemd/$SERVICE_NAME.service" ]; then
        log_error "Systemd service file not found"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build the application
build_application() {
    log_info "Building application..."
    
    # Clean build
    go clean
    
    # Build for Linux (in case we're cross-compiling)
    GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $APP_NAME cmd/main.go
    
    # Verify binary
    if [ ! -f "$APP_NAME" ]; then
        log_error "Failed to build application"
        exit 1
    fi
    
    log_success "Application built successfully ($(du -h $APP_NAME | cut -f1))"
}

# Create deployment package
create_package() {
    log_info "Creating deployment package..."
    
    # Create temporary directory for packaging
    TEMP_DIR=$(mktemp -d)
    PACKAGE_DIR="$TEMP_DIR/salam-monitoring"
    
    mkdir -p "$PACKAGE_DIR/config"
    mkdir -p "$PACKAGE_DIR/systemd"
    mkdir -p "$PACKAGE_DIR/scripts"
    
    # Copy files
    cp "$APP_NAME" "$PACKAGE_DIR/"
    cp config/config.yaml "$PACKAGE_DIR/config/"
    cp systemd/$SERVICE_NAME.service "$PACKAGE_DIR/systemd/"
    cp deploy.sh "$PACKAGE_DIR/scripts/"
    
    # Create deployment archive
    cd "$TEMP_DIR"
    tar -czf "salam-monitoring-$(date +%Y%m%d-%H%M%S).tar.gz" salam-monitoring/
    
    # Move package to current directory
    mv "salam-monitoring-$(date +%Y%m%d-%H%M%S).tar.gz" "$OLDPWD/"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    
    log_success "Deployment package created"
}

# Deploy to target server
deploy_to_server() {
    if [ "$TARGET_HOST" = "localhost" ] || [ "$TARGET_HOST" = "local" ]; then
        deploy_local
    else
        deploy_remote
    fi
}

# Local deployment
deploy_local() {
    log_info "Deploying to local server..."
    
    # Stop service if running
    if systemctl is-active --quiet $SERVICE_NAME; then
        log_info "Stopping $SERVICE_NAME service..."
        sudo systemctl stop $SERVICE_NAME
    fi
    
    # Backup existing installation
    if [ -d "$DEPLOY_DIR" ]; then
        log_info "Backing up existing installation..."
        sudo mv "$DEPLOY_DIR" "$BACKUP_DIR"
    fi
    
    # Create deployment directory
    sudo mkdir -p "$DEPLOY_DIR"
    sudo mkdir -p "$DEPLOY_DIR/data"
    sudo mkdir -p "$DEPLOY_DIR/logs"
    
    # Copy files
    sudo cp "$APP_NAME" "$DEPLOY_DIR/"
    sudo cp config/config.yaml "$DEPLOY_DIR/config/"
    
    # Set permissions
    sudo chown -R informaticaadmin:informaticaadmin "$DEPLOY_DIR"
    sudo chmod +x "$DEPLOY_DIR/$APP_NAME"
    
    # Install systemd service
    sudo cp "systemd/$SERVICE_NAME.service" "/etc/systemd/system/"
    sudo systemctl daemon-reload
    
    # Enable and start service
    sudo systemctl enable $SERVICE_NAME
    sudo systemctl start $SERVICE_NAME
    
    # Check service status
    if systemctl is-active --quiet $SERVICE_NAME; then
        log_success "Service $SERVICE_NAME is running"
    else
        log_error "Service $SERVICE_NAME failed to start"
        sudo journalctl -u $SERVICE_NAME --no-pager -n 20
        exit 1
    fi
    
    log_success "Local deployment completed successfully"
}

# Remote deployment
deploy_remote() {
    log_info "Deploying to remote server: $TARGET_HOST"
    
    # Check SSH connectivity
    if ! ssh -o ConnectTimeout=5 "$TARGET_HOST" "echo 'Connection test'" > /dev/null 2>&1; then
        log_error "Cannot connect to $TARGET_HOST via SSH"
        exit 1
    fi
    
    # Create deployment package
    PACKAGE_NAME="salam-monitoring-$(date +%Y%m%d-%H%M%S).tar.gz"
    create_package
    
    # Transfer package
    log_info "Transferring package to $TARGET_HOST..."
    scp "$PACKAGE_NAME" "$TARGET_HOST:/tmp/"
    
    # Execute remote deployment
    ssh "$TARGET_HOST" << EOF
set -e

# Extract package
cd /tmp
tar -xzf $PACKAGE_NAME
cd salam-monitoring

# Run deployment script
chmod +x scripts/deploy.sh
sudo scripts/deploy.sh local

# Cleanup
cd /
rm -rf /tmp/salam-monitoring /tmp/$PACKAGE_NAME
EOF
    
    # Cleanup local package
    rm -f "$PACKAGE_NAME"
    
    log_success "Remote deployment completed successfully"
}

# Health check
health_check() {
    log_info "Performing health check..."
    
    sleep 5  # Wait for service to fully start
    
    # Check if service is running
    if ! systemctl is-active --quiet $SERVICE_NAME; then
        log_error "Service is not running"
        return 1
    fi
    
    # Check if HTTP endpoint is responding
    if command -v curl > /dev/null; then
        if curl -f http://localhost:8080 > /dev/null 2>&1; then
            log_success "HTTP endpoint is responding"
        else
            log_warning "HTTP endpoint is not responding"
        fi
    fi
    
    # Show service status
    systemctl status $SERVICE_NAME --no-pager
    
    log_success "Health check completed"
}

# Show service logs
show_logs() {
    log_info "Recent service logs:"
    sudo journalctl -u $SERVICE_NAME --no-pager -n 50
}

# Main deployment flow
main() {
    log_info "Starting deployment of Salam Monitoring Platform"
    
    check_prerequisites
    
    if [ "$TARGET_HOST" = "localhost" ] || [ "$TARGET_HOST" = "local" ]; then
        build_application
    fi
    
    deploy_to_server
    
    if [ "$TARGET_HOST" = "localhost" ] || [ "$TARGET_HOST" = "local" ]; then
        health_check
        show_logs
    fi
    
    log_success "Deployment completed successfully!"
    log_info "Access the monitoring platform at: http://$TARGET_HOST:8080"
}

# Handle script arguments
case "${2:-deploy}" in
    "build")
        check_prerequisites
        build_application
        log_success "Build completed! Binary: $APP_NAME ($(file $APP_NAME | cut -d: -f2-))"
        ;;
    "package")
        check_prerequisites
        build_application
        create_package
        ;;
    "health")
        health_check
        ;;
    "logs")
        show_logs
        ;;
    "deploy"|*)
        main
        ;;
esac