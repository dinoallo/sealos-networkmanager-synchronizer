package store

type TagProperty struct {
	Name            string `bson:"name"` // pk
	CurSentByteMark uint64 `bson:"cur_sent_byte_mark"`
	CurRecvByteMark uint64 `bson:"cur_recv_byte_mark"`
	SentBytes       uint64 `bson:"sent_bytes"`
	RecvBytes       uint64 `bson:"recv_bytes"`
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
}

type PortFeed struct {
	ID                string                     `bson:"pf_id"`
	Prop              PortFeedProp               `bson:"pf_prop"`
	AddressProperties map[string]AddressProperty `bson:"address_properties"`
}

func (pta *PodTrafficAccount) GetByteMark(addr string, tag string, t int, isAddrEncoded bool, byteMark *uint64) error {
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
						*byteMark = pta.AddressProperties[id].TagProperties[tag].CurRecvByteMark
					} else if t == 1 {
						*byteMark = pta.AddressProperties[id].TagProperties[tag].CurSentByteMark
					}
				}
			}
		}
	}
	return nil
}

func (pf *PortFeed) GetByteMark(addr string, tag string, t int, isAddrEncoded bool, byteMark *uint64) error {
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
	if pf.AddressProperties != nil {
		if _, ok := pf.AddressProperties[id]; ok {
			if pf.AddressProperties[id].TagProperties != nil {
				if _, found := pf.AddressProperties[id].TagProperties[tag]; found {
					if t == 0 {
						*byteMark = pf.AddressProperties[id].TagProperties[tag].CurRecvByteMark
					} else if t == 1 {
						*byteMark = pf.AddressProperties[id].TagProperties[tag].CurSentByteMark
					}
				}
			}
		}
	}
	return nil
}

func (pta *PodTrafficAccount) GetBytes(addr string, tag string, t int, isAddrEncoded bool, bytes *uint64) error {
	if bytes == nil {
		return nil
	}
	*bytes = 0
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
						*bytes = pta.AddressProperties[id].TagProperties[tag].RecvBytes
					} else if t == 1 {
						*bytes = pta.AddressProperties[id].TagProperties[tag].SentBytes
					}
				}
			}
		}
	}
	return nil
}

func (pta *PodTrafficAccount) GetTagProperty(addr string, tag string, isAddrEncoded bool, tp *TagProperty) error {
	if tp == nil {
		return nil
	}
	*tp = TagProperty{
		SentBytes:       0,
		RecvBytes:       0,
		CurSentByteMark: 0,
		CurRecvByteMark: 0,
	}
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
					*tp = pta.AddressProperties[id].TagProperties[tag]
				}
			}
		}
	}
	return nil
}

func (pf *PortFeed) GetTagProperty(addr string, tag string, isAddrEncoded bool, tp *TagProperty) error {
	if tp == nil {
		return nil
	}
	*tp = TagProperty{
		SentBytes:       0,
		RecvBytes:       0,
		CurSentByteMark: 0,
		CurRecvByteMark: 0,
	}
	var id string = addr
	if !isAddrEncoded {
		if err := encodeIP(addr, &id); err != nil {
			return err
		}
	}
	if pf.AddressProperties != nil {
		if _, ok := pf.AddressProperties[id]; ok {
			if pf.AddressProperties[id].TagProperties != nil {
				if _, found := pf.AddressProperties[id].TagProperties[tag]; found {
					*tp = pf.AddressProperties[id].TagProperties[tag]
				}
			}
		}
	}
	return nil
}
