#!/bin/bash
set -e

# Set up environment variables for LocalStack S3
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
export S3_ENDPOINT=http://localhost:4566

# Configure obscure to use LocalStack S3
obscure provider add s3 <<EOF
ci-test-bucket
us-east-1
test
test
$S3_ENDPOINT
EOF

# Enable the provider as default and active
obscure switch-provider --default s3

# Create the bucket in LocalStack
aws --endpoint-url=$S3_ENDPOINT s3 mb s3://ci-test-bucket || true

# Create a test file
TEST_FILE=testfile.txt
echo "hello world" > $TEST_FILE

# Run backup
obscure backup $TEST_FILE --tag=ci-test --version=1.0 --direct

# Check if file exists in S3 (via AWS CLI)
aws --endpoint-url=$S3_ENDPOINT s3 ls s3://ci-test-bucket/backups/ | grep ci-test || (echo "Backup not found in S3" && exit 1)

# Run rm
obscure rm ci-test/1.0_ci-test.tar

# Check that file is gone
! aws --endpoint-url=$S3_ENDPOINT s3 ls s3://ci-test-bucket/backups/ | grep ci-test

echo "Integration test passed!" 