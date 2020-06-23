#!/bin/bash
FAKENET_DIR=~/.lachesis/fakenet-1
FAKENET_KEYSTOR=$FAKENET_DIR/keystore
MAINNET_KEYSTOR=/home/user/.lachesis/keystore
if [ ! -d "$FAKENET_DIR" ]; then
  ./build/lachesis --fakenet 1/1 --rpc --rpcapi="eth,debug,admin,web3,personal,net,txpool,ftm,sfc" --ws --wsorigins="*" --wsapi="eth,debug,admin,web3,personal,net,txpool,ftm,sfc" --config=config.toml
fi

NUMBER_KEYS=$(ls -1 "$FAKENET_KEYSTOR" | wc -l)
if [[ $NUMBER_KEYS < 2 ]]; then
  ./build/lachesis account new
  ./build/lachesis account new
  cp $MAINNET_KEYSTOR/* $FAKENET_KEYSTOR
  echo "added key to $FAKENET_KEYSTOR"
fi
./build/lachesis --fakenet 1/1 --rpc --rpcapi="eth,debug,admin,web3,personal,net,txpool,ftm,sfc" --ws --wsorigins="*" --wsapi="eth,debug,admin,web3,personal,net,txpool,ftm,sfc" --config=config.toml
