# Gagarin.Network
## FAQ 
[Go to link](FAQ.md)
## Network
Network expose 2 different ports 9080 for network communication between peers and 9180 for websocket communication
### Common shell commands
* generates protobuf classes
    * ```make protos```
* build gnetwork binaries
    * ```make build```
* generate mock classes
    * ```mockery --all```
* start network
    * ```gnetwork -l $i, where i=0...N```
    
### Docker
Common way to start network cluster is via Docker.
* create local network 
    * ```docker network create --driver=bridge --subnet=172.18.0.0/16 gagarin	```
* build docker image
    * ```docker build .```
* run container ($1 is node index, $2 is image short hash)
    * ```docker run --net gagarin --ip 172.18.0.1$1 -it -e GN_IND=$1 -p 918$1:9180 $2 ./gnetwork``` 

### Static dir
* peers.json
```
{
	"peers": [ #list of bootstrap peers
		{
			"addr": "0xDd9811Cfc24aB8d56036A8ecA90C7B8C75e35950", #blockchain addr
			"pub": "8d00d7cea8f1f6edc961fe2867ffb94a1e78ef8b8eb416d6f5b7b3f6dd9811cfc24ab8d56036a8eca90c7b8c75e35950", # blockchain G1 public key
			"MultiAddress": "/ip4/172.18.0.10/tcp/9080/p2p/16Uiu2HAmGgeX9Sr75ofG4rQbUhRiUH2AuKii5QCdD9h8NT83afo4" #libp2p multiaddr
		},
  ...
```
* peer(i).json, where i is the same i from start network
```
{
	"addr": "0xA60c85Be9c2b8980Dbd3F82cF8CE19DE12f3E829", #blockchain addr
	"id": "16Uiu2HAmCRVJB9iKxon89fp8J25KvEgkEmgViLnaYi7A9aiZoSpy", #libp2p peer id (public ECDSA key)
	"pk": "4a157ef4295be2413c59613cd5dd4f3b0688ac6227709a9634608f33a49250ef", #blockchain private key 
	"pkpeer": "080212208ceba2eb4883bfd2638dd37807b9986ea9ce18319b4bcf663fafb007db3087e6" #libp2p peer private ECDSA key
	"pub": "8d00d7cea8f1f6edc961fe2867ffb94a1e78ef8b8eb416d6f5b7b3f6dd9811cfc24ab8d56036a8eca90c7b8c75e35950" #blockchain pub key

}
```
* seed.json starting state
```
{
  "accounts": [ #account list
    {
      "address": "0xA60c85Be9c2b8980Dbd3F82cF8CE19DE12f3E829", #blockchain address
      "balance": 1000 #starting balance
    },
    ...
```
* settings.yaml
```
Hotstuff: #hotstuff settings 
  N: 10 #total node amount, must be 3 * n + 1, here n = 3
  Delta: 5000 #stage duration in milliseconds, 2*Delta is v iew change duration, 4* Delta is GST
  BlockDelta: 10 #max time in millisecinds to wait for transactions 
Network:
  MinPeerThreshold: 3 #minimum peer count to bootstrap connection, from bootstrap list
  ReconnectPeriod: 10000 #period after watchdog checks bootstrap connections and reconnects whether count less then MinPeerThreshold
  ConnectionTimeout: 3000 # timeout
Storage:
  Dir: /var/folders/m7/c56_pk2n0dj4xbplcndc9d140000gn/T/ #storage dir
```
## Tests
There are several tests in the project. We have very light unit tests which are placed near scripts to test and integration tests in the network/test directory.
Main integration tests are in it_test.go. Each test implements specific scenario and runs main hotstuff processes. Important: for now we mock block service and tx services and they are not covered in it_tests, tests for these services can be found in block_protocol_test.go and tx_service_test.go.
