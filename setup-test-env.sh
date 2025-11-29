#!/bin/bash

# Test setup script for Salam Monitoring Platform

echo "🔄 Setting up test environment for Salam Monitoring Platform..."

# Create additional test data directories
echo "📁 Creating additional test directories..."

# Create yesterday's data for testing date filtering
YESTERDAY=$(date -d "yesterday" +"%Y-%m-%d")
mkdir -p "nfs_backup/monitoring/miniboss/$YESTERDAY/old_workflow"

# Create more test workflows for today
TODAY=$(date +"%Y-%m-%d")
mkdir -p "nfs_backup/monitoring/miniboss/$TODAY/spark_etl_job"
mkdir -p "nfs_backup/monitoring/platform1/$TODAY/batch_processing"

echo "📝 Creating sample log files..."

# Create more sample logs
cat > "nfs_backup/monitoring/miniboss/$TODAY/spark_etl_job/info.log" << 'EOF'
2025-11-20 14:00:00 INFO: Starting Spark ETL job
2025-11-20 14:00:05 INFO: Spark context initialized
2025-11-20 14:00:10 INFO: Reading input data from HDFS
2025-11-20 14:02:30 INFO: Processing 5.2M records
2025-11-20 14:05:45 INFO: Applying data transformations
2025-11-20 14:08:20 INFO: Writing results to data lake
2025-11-20 14:10:15 INFO: Job completed successfully
EOF

cat > "nfs_backup/monitoring/miniboss/$TODAY/spark_etl_job/run.log" << 'EOF'
WORKFLOW: spark_etl_job
START_TIME: 2025-11-20 14:00:00
END_TIME: 2025-11-20 14:10:15
STATUS: COMPLETED
RECORDS_PROCESSED: 5200000
DURATION: 10m15s
SPARK_VERSION: 3.4.0
EXECUTORS: 8
MEMORY_PER_EXECUTOR: 4GB
CORES_PER_EXECUTOR: 2
EOF

# Create a workflow that's still running (no run.log)
cat > "nfs_backup/monitoring/platform1/$TODAY/batch_processing/info.log" << 'EOF'
2025-11-20 15:30:00 INFO: Starting batch processing workflow
2025-11-20 15:30:05 INFO: Initializing batch processor
2025-11-20 15:30:10 INFO: Loading configuration files
2025-11-20 15:32:00 INFO: Processing batch 1 of 10
2025-11-20 15:35:30 INFO: Processing batch 2 of 10
2025-11-20 15:38:45 INFO: Processing batch 3 of 10
2025-11-20 15:42:10 INFO: Processing batch 4 of 10
EOF

# Create old workflow with errors
cat > "nfs_backup/monitoring/miniboss/$YESTERDAY/old_workflow/info.log" << 'EOF'
2025-11-19 16:00:00 INFO: Starting old workflow
2025-11-19 16:05:00 INFO: Workflow completed
EOF

cat > "nfs_backup/monitoring/miniboss/$YESTERDAY/old_workflow/run.log" << 'EOF'
WORKFLOW: old_workflow
START_TIME: 2025-11-19 16:00:00
END_TIME: 2025-11-19 16:05:00
STATUS: COMPLETED
RECORDS_PROCESSED: 100000
DURATION: 5m00s
EOF

echo "🔧 Creating test configuration..."

# Update config to use test mode
cat > config/config.yaml << 'EOF'
mode: "test"

server:
  port: 8080

paths:
  nfs_root_test: "./nfs_backup/monitoring"
  nfs_root_prod: "/home/informaticaadmin/nfs_backup/monitoring"

services:
  yarn_rm_url: "http://localhost:8088"
  informatica_db:
    host: "localhost"
    service: "TESTDB"
    user: "test_user"
    password: "test_pass"

database:
  sqlite_path: "data/history.db"
EOF

# Create data directory
mkdir -p data

echo "🧪 Running basic tests..."

# Test configuration loading
if [ -f "config/config.yaml" ]; then
    echo "✅ Configuration file created"
else
    echo "❌ Configuration file missing"
fi

# Test directory structure
if [ -d "nfs_backup/monitoring/miniboss/$TODAY/data_ingestion_workflow" ]; then
    echo "✅ Test NFS structure created"
else
    echo "❌ Test NFS structure missing"
fi

# Count log files
LOG_COUNT=$(find nfs_backup/monitoring -name "*.log" | wc -l)
echo "📊 Created $LOG_COUNT test log files"

# Test data summary
echo "📈 Test Data Summary:"
echo "   - Sources: $(ls nfs_backup/monitoring | wc -l)"
echo "   - Today's workflows: $(find nfs_backup/monitoring -path "*/$TODAY/*" -type d | wc -l)"
echo "   - Failed workflows: $(find nfs_backup/monitoring -name "error.log" | wc -l)"
echo "   - Completed workflows: $(find nfs_backup/monitoring -name "run.log" | wc -l)"

echo "🚀 Test environment setup complete!"
echo ""
echo "You can now:"
echo "   1. Build the application: go build -o salam-monitor cmd/main.go"
echo "   2. Run in test mode: ./salam-monitor --mode=test"
echo "   3. Open browser to: http://localhost:8080"
echo ""
echo "CLI examples:"
echo "   ./salam-monitor logs today"
echo "   ./salam-monitor --config=config/config.yaml"