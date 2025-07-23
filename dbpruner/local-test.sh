#!/bin/bash

set -euo pipefail

if ! oc get svc mongodb > /dev/null 2>&1; then
    echo "MongoDB service not found"
    echo "Make sure you are using a kubeconfig that has access to the running cluster and has the ci-testgrid namespace"
    exit 1
fi

oc port-forward svc/mongodb 27017:27017 &

# Set the MongoDB URI and database name
export MONGODB_URI="mongodb://localhost:27017"
export DRY_RUN=true

go run main.go