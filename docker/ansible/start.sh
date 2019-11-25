#!/bin/bash

echo "Run $1 instances..."
ansible-playbook ./deploy-lachesis.yaml -i ./hosts-$1 -v
