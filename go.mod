module github.com/gagarinchain/network

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/emirpasic/gods v1.12.0
	github.com/ethereum/go-ethereum v1.9.15 // indirect
	github.com/ferranbt/fastssz v0.0.0-20200826142241-3a913c5a1313 // indirect
	github.com/gagarinchain/common v0.1.14
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/ipfs/go-cid v0.0.5
	github.com/ipfs/go-datastore v0.4.4
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-log v1.0.4
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99
	github.com/libp2p/go-libp2p v0.8.3
	github.com/libp2p/go-libp2p-core v0.5.6
	github.com/libp2p/go-libp2p-discovery v0.4.0
	github.com/libp2p/go-libp2p-kad-dht v0.7.11
	github.com/libp2p/go-libp2p-peerstore v0.2.4 // indirect
	github.com/libp2p/go-libp2p-pubsub v0.2.7
	github.com/magiconair/properties v1.8.1
	github.com/minio/highwayhash v1.0.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.2.2
	github.com/multiformats/go-multihash v0.0.13
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/phoreproject/bls v0.0.0-20200525203911-a88a5ae26844
	github.com/pkg/errors v0.9.1
	github.com/protolambda/zssz v0.1.5 // indirect
	github.com/prysmaticlabs/go-bitfield v0.0.0-20200618145306-2ae0807bef65 // indirect
	github.com/prysmaticlabs/go-ssz v0.0.0-20200612203617-6d5c9aa213ae
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/status-im/keycard-go v0.0.0-20190316090335-8537d3370df4
	github.com/stretchr/testify v1.5.1
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	golang.org/x/net v0.0.0-20200425230154-ff2c4b7c35a0
	google.golang.org/grpc v1.20.1
	gopkg.in/yaml.v2 v2.3.0
)

go 1.14

//replace github.com/gagarinchain/common => ../common
