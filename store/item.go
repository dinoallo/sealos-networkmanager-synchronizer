package store

type TagProperty struct {
	Name      string `bson:"name"` // pk
	SentBytes uint64 `bson:"sent_bytes"`
	RecvBytes uint64 `bson:"recv_bytes"`
}
type AddressProperty struct {
	Address       string                 `bson:"address"` // pk
	TagProperties map[string]TagProperty `bson:"tag_properties"`
}

type PodTrafficAccount struct {
	NamespacedName    string                     `bson:"namespaced_name"` // pk
	Name              string                     `bson:"name"`
	Namespace         string                     `bson:"namespace"`
	AddressProperties map[string]AddressProperty `bson:"address_properties"`
}
