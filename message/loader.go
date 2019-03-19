package message

type CommitteeLoader interface {
	LoadFromFile() []*Peer
}
