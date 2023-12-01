package store

type TagProperty struct {
	Name             string `bson:"name"` // pk
	LastSentByteMark uint64 `bson:"last_sent_byte_mark"`
	LastRecvByteMark uint64 `bson:"last_recv_byte_mark"`
	CurSentByteMark  uint64 `bson:"cur_sent_byte_mark"`
	CurRecvByteMark  uint64 `bson:"cur_recv_byte_mark"`
	SentBytes        uint64 `bson:"sent_bytes"`
	RecvBytes        uint64 `bson:"recv_bytes"`
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

type TagPropReq struct {
	NamespacedName string
	Addr           string
	Tag            string
}

type PortFeedProp struct {
	Namespace string `bson:"namespace"`
	Pod       string `bson:"pod"`
	Port      int32  `bson:"port"`
}

type PortFeed struct {
	ID        string       `bson:"pf_id"`
	Prop      PortFeedProp `bson:"pf_prop"`
	SentBytes uint64       `bson:"sent_bytes"`
	RecvBytes uint64       `bson:"recv_bytes"`
}

func (pta *PodTrafficAccount) GetByteMark(addr string, tag string, t int, getLast bool, isAddrEncoded bool, byteMark *uint64) error {
	if byteMark == nil {
		return nil
	}
	*byteMark = 0
	var id string = addr
	if !isAddrEncoded {
		if err := encodeIP(addr, &id); err != nil {
			return err
		}
	}
	if pta.AddressProperties != nil {
		if _, ok := pta.AddressProperties[id]; ok {
			if pta.AddressProperties[id].TagProperties != nil {
				if _, found := pta.AddressProperties[id].TagProperties[tag]; found {
					if t == 0 {
						if getLast {
							*byteMark = pta.AddressProperties[id].TagProperties[tag].LastRecvByteMark
						} else {
							*byteMark = pta.AddressProperties[id].TagProperties[tag].CurRecvByteMark
						}
					} else if t == 1 {
						if getLast {
							*byteMark = pta.AddressProperties[id].TagProperties[tag].LastSentByteMark
						} else {
							*byteMark = pta.AddressProperties[id].TagProperties[tag].CurSentByteMark
						}
					}
				}
			}
		}
	}
	return nil
}
