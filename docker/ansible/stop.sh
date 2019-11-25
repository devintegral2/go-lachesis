#!/bin/bash

echo "Stop all instances..."
ansible-playbook ./stop-lachesis.yaml -i ./hosts
