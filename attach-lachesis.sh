#!/bin/bash
./build/lachesis --jspath "." --exec 'loadScript("deploy_sfc.js")' attach ~/.lachesis/fakenet-1/lachesis.ipc
