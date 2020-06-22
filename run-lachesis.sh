#!/bin/bash
./build/lachesis --fakenet 1/1 --rpc --rpcapi="eth,debug,admin,web3,personal,net,txpool,ftm,sfc" --ws --wsorigins="*" --wsapi="eth,debug,admin,web3,personal,net,txpool,ftm,sfc" --config=config.toml
