### Deploy lachesis to servers over ansible

#### Description

Configuration used for deploy lachesis in any count of servers with posibility of run several instances on every servers.

#### Hosts configurations example

```text
192.168.88.10 start_node_num=1 last_node_num=1 all_nodes_count=3
192.168.88.25 start_node_num=2 last_node_num=2 all_nodes_count=3
192.168.88.26 start_node_num=3 last_node_num=3 all_nodes_count=3
```

Main values:
* all_nodes_count - count of all lachesis nodes in current cluster
* start_node_num - number of lachesis first instance at this host
* last_node_num - number of lachesis last instance at this host

If you want run only one lachesis instance in one host, use value `start_node_num` equal value `last_node_num`.

Other example:
```text
192.168.88.10 start_node_num=1 last_node_num=3 all_nodes_count=21
192.168.88.25 start_node_num=4 last_node_num=6 all_nodes_count=21
192.168.88.26 start_node_num=7 last_node_num=9 all_nodes_count=21
192.168.88.28 start_node_num=10 last_node_num=12 all_nodes_count=21
192.168.88.29 start_node_num=13 last_node_num=15 all_nodes_count=21
192.168.88.36 start_node_num=16 last_node_num=17 all_nodes_count=21
192.168.88.61 start_node_num=18 last_node_num=19 all_nodes_count=21
192.168.88.80 start_node_num=20 last_node_num=21 all_nodes_count=21
``` 
In this config runing 21 lachesis instances on 8 hosts:
* 3 instances (1-3) on host 192.168.88.10
* 3 instances (4-6) on host 192.168.88.25
* 3 instances (7-9) on host 192.168.88.26
* 3 instances (10-12) on host 192.168.88.28
* 3 instances (13-15) on host 192.168.88.29
* 2 instances (16-17) on host 192.168.88.36
* 2 instances (18-19) on host 192.168.88.61
* 2 instances (20-21) on host 192.168.88.80

RPC port calculated like `18545 + <instance number>`. For example, instance 13 will have rpc address for query: `192.168.88.29:18558`. Instance 21 will have rpc address for query: `192.168.88.80:18566`.  

#### Scripts

Script `start.sh` use prepared hosts configs named like `hosts-<N>`, where `N` is count of lachesis instances. If you want run 10 insances on this hosts list, you can run `./start.sh 10`. Use script `./stop.sh` without parameters for stop all instances at all hosts.
