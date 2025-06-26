#!/bin/bash
set -e

# Set up environment variables for fake-gcs-server
export GCS_PROJECT_ID=fake-project
export GCS_BUCKET=ci-test-bucket
export GCS_ENDPOINT=http://localhost:4443
export GCS_SERVICE_ACCOUNT=fake-service-account.json

# Create a fake service account key file (fake-gcs-server doesn't validate it)
echo '{"type": "service_account", "project_id": "fake-project"}' > $GCS_SERVICE_ACCOUNT

# Create the bucket in fake-gcs-server
gsutil -o "GSUtil:default_project_id=$GCS_PROJECT_ID" -o "Credentials:gs_service_key_file=$GCS_SERVICE_ACCOUNT" mb -p $GCS_PROJECT_ID -l US-EAST1 $GCS_ENDPOINT/storage/v1/b/$GCS_BUCKET || true

# Configure obscure to use fake-gcs-server GCS
obscure provider add gcs <<EOF
$GCS_PROJECT_ID
$GCS_SERVICE_ACCOUNT
EOF

# Enable the provider as default and active
obscure switch-provider --default gcs

# Create a test file
TEST_FILE=testfile.txt
echo "hello world" > $TEST_FILE

# Run backup
obscure backup $TEST_FILE --tag=ci-test --version=1.0 --direct

# Check if file exists in GCS (via gsutil)
gsutil -o "GSUtil:default_project_id=$GCS_PROJECT_ID" -o "Credentials:gs_service_key_file=$GCS_SERVICE_ACCOUNT" ls $GCS_ENDPOINT/storage/v1/b/$GCS_BUCKET/o | grep ci-test || (echo "Backup not found in GCS" && exit 1)

# Run rm
obscure rm ci-test/1.0_ci-test.tar

# Check that file is gone
! gsutil -o "GSUtil:default_project_id=$GCS_PROJECT_ID" -o "Credentials:gs_service_key_file=$GCS_SERVICE_ACCOUNT" ls $GCS_ENDPOINT/storage/v1/b/$GCS_BUCKET/o | grep ci-test

echo "GCS integration test passed!" 