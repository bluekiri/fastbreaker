#!/bin/bash
GOFMT_OUTPUT="$(gofmt -d . 2>&1)"
if [ -n "$GOFMT_OUTPUT" ]; then 
    echo "Please format your code."
    echo "${GOFMT_OUTPUT}"
    exit 1
fi