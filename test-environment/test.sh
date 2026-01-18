#!/bin/bash
# Test script for nigel

# Clean up
rm nigel/demo-task/*.log
rm .fixed-item-*

# Check for --inactivity-test flag
if [[ "$1" == "--inactivity-test" ]]; then
    # Simulate long delays to test the 30-second inactivity timer
    echo "Testing inactivity timer (will pause for 35s between messages)..."
    MOCK_CLAUDE_INACTIVITY_TEST=1 nigel demo-task
else
    # Normal test run
    nigel demo-task
fi
