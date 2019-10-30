# send_transactions_flow

This application generate random transfer transactions on nodes list (validators) between random generated accounts. Application use for performance test with real nodes.

## Parameters

`- nodes` - file name with nodes URL list. For example, like text file:
```text
http://localhost:18546
http://localhost:18547
http://localhost:18548
http://localhost:18549
http://localhost:18550
``` 
`- acc` - count of random generated accounts. It will be used for transfers.

`-donor_addr` - address of donor account ('0x...' format). This account will be used for create started balance on test accounts. Donor account should be contain great then `100000000 * acc` balance.

`-donor_key` - private key of donor account (raw hex format without '0x').

`- mode` - mode of waiting confirm transactions:
* 0 - not wait and not check confirm transactions;
* 1 - not wait, but check count of confirmed transactions after send transactions batch;
* 2 - wait for confirm all transaction after send transactions batch.

`-trx_batch` - count of transactions in batch.

`-trx_count` - limit transactions count for test. If this param not set - test work in infinity mode. If used this parameters - only send this count of transaction to each mode and finish.

`-max_flow` - limit of flow speed for sending transaction on each node (transactions/minute).

`-rand_flow` - random factor for create different transactions flow speed to different nodes. For node set flow speed limit as `max_flow +- rand_flow/2`.

## Output

`ok` - count of sended transactions (in parentheses - count of processed transactions: confirmed, missed or get timeout);

`errors` - count of transactions, get error when try send;

`missed` - count of transactions, sended without errors, but not detected on node after some time;

`timeouts` - count of transactions, not confirmed long time (1 minute) after send;

`pending` - count of transactions, wait of confirmation;

Speed of transactions `tx/min` - speed of sending transactions;
