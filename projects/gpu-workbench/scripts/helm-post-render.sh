#!/bin/sh
# Local single-node lab: bounded retry after asynchronous device cache updates.
# No network-topology CR is needed for this single-node experiment.
sed 's/--enable-network-topology-manager=true/--enable-network-topology-manager=false\
            - --pod-max-in-unschedulable-pods-duration=10s/'
