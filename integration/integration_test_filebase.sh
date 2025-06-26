#!/bin/bash
set -e

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
export S3_ENDPOINT=http://localhost:4566

# Configure obscure to use S3-compatible (Filebase-like) provider
obscure provider add s3-compatible <<EOF
ci-test-filebase
us-east-1
test
test
$S3_ENDPOINT
FilebaseTest
EOF

obscure switch-provider --default s3-compatible

aws --endpoint-url=$S3_ENDPOINT s3 mb s3://ci-test-filebase || true

TEST_FILE=testfile.txt
echo "hello world" > $TEST_FILE

obscure backup $TEST_FILE --tag=ci-test --version=1.0 --direct

aws --endpoint-url=$S3_ENDPOINT s3 ls s3://ci-test-filebase/backups/ | grep ci-test || (echo "Backup not found in S3-compatible" && exit 1)

obscure rm ci-test/1.0_ci-test.tar

! aws --endpoint-url=$S3_ENDPOINT s3 ls s3://ci-test-filebase/backups/ | grep ci-test

echo "S3-compatible (Filebase) integration test passed!" 