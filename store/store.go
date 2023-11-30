package store

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/sqids/sqids-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	SQID_ALPHABET = "abcdefghijklmnopqrstuvwxyz0123456789"
)

type DBCred struct {
	DBHost string
	DBPort string
	DBUser string
	DB     string
	DBPass string
}

type Store struct {
	Cred     *DBCred
	Log      *logr.Logger
	db       *mongo.Database
	dbClient *mongo.Client
}

func (s *Store) Launch(ctx context.Context) error {
	if s.Cred == nil || s.Log == nil {
		return fmt.Errorf("the credential and the logger shouldn't be nil")
	}
	cred := s.Cred
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/?maxPoolSize=20&w=majority", cred.DBUser, cred.DBPass, cred.DBHost, cred.DBPort)
	clientOps := options.Client().ApplyURI(uri)
	if client, err := mongo.Connect(ctx, clientOps); err != nil {
		return err
	} else {
		s.dbClient = client
	}
	if err := s.dbClient.Ping(ctx, readpref.Primary()); err != nil {
		return err
	} else {
		s.db = s.dbClient.Database(cred.DB)
	}
	return nil
}
func (s *Store) Close(ctx context.Context) {
	s.dbClient.Disconnect(context.TODO())
}

func (s *Store) FindPTA(ctx context.Context, nn string, pta *PodTrafficAccount) (bool, error) {
	if pta == nil {
		return false, fmt.Errorf("the pta cannot be nil")
	}
	if s.db == nil {
		return false, fmt.Errorf("please call Launch first")
	}
	coll := s.db.Collection("pod_traffic_accounts")
	filter := bson.D{
		{
			Key:   "namespaced_name",
			Value: nn,
		},
	}
	getCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err := coll.FindOne(getCtx, filter).Decode(pta); err != nil {
		if err != mongo.ErrNoDocuments {
			return false, err
		} else {
			return false, nil
		}
	}
	return true, nil
}

func (s *Store) UpdateFieldUint64(ctx context.Context, req TagPropReq, op string, field string, value uint64) error {
	log := s.Log
	if log == nil {
		return nil
	}
	if s.db == nil {
		return fmt.Errorf("please call Launch first")
	}
	coll := s.db.Collection("pod_traffic_accounts")
	updateCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	filter := bson.D{{
		Key:   "namespaced_name",
		Value: req.NamespacedName,
	}}
	var id string
	if err := encodeIP(req.Addr, &id); err != nil {
		return err
	}
	key := fmt.Sprintf("address_properties.%s.tag_properties.%s.%s", id, req.Tag, field)
	update := bson.D{{
		Key: op,
		Value: bson.D{{
			Key:   key,
			Value: value,
		}},
	}}
	if _, err := coll.UpdateOne(updateCtx, filter, update, opts); err != nil {
		return err
	} else {
		log.Info("the data has been updated", "field", key)
	}
	return nil
}
func (s *Store) IncPFByteField(ctx context.Context, req PortFeedProp, field string, value uint64) error {
	log := s.Log
	if log == nil {
		return nil
	}
	if s.db == nil {
		return fmt.Errorf("please call Launch first")
	}
	coll := s.db.Collection("pod_traffic_accounts")
	updateCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	var id string
	if err := encodeIP(req.Addr, &id); err != nil {
		return err
	}
	pf_id := fmt.Sprintf("%s-%s-%s-%s", req.Namespace, req.Pod, id, fmt.Sprint(req.Port))
	filter := bson.D{{
		Key:   "pf_id",
		Value: pf_id,
	}}
	update := bson.D{{
		Key: "$inc",
		Value: bson.D{{
			Key:   field,
			Value: value,
		}},
	}}
	if _, err := coll.UpdateOne(updateCtx, filter, update, opts); err != nil {
		return err
	} else {
		log.Info("the data of the port feed has been updated", "field", field)
	}
	return nil
}
func (s *Store) Save(ctx context.Context, key string, pta *PodTrafficAccount) error {
	if s.db == nil {
		return fmt.Errorf("please call Launch first")
	}
	coll := s.db.Collection("pod_traffic_accounts")
	putCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	opts := options.Replace().SetUpsert(true)
	filter := bson.D{{
		Key:   "namespaced_name",
		Value: key,
	}}
	replacement := pta
	if _, err := coll.ReplaceOne(putCtx, filter, replacement, opts); err != nil {
		return err
	}
	return nil
}

func encodeIP(_ipAddr string, id *string) error {
	if id == nil {
		return fmt.Errorf("id shouldn't be nil")
	}
	var numbers []uint64
	_ip := net.ParseIP(_ipAddr)
	if _ip == nil {
		return fmt.Errorf("this is not a valid ip address")
	}
	if IsIPv4(_ip.String()) {
		if ip, err := IPv4ToInt(_ip); err != nil {
			return err
		} else {
			numbers = []uint64{uint64(ip)}
		}

	} else if IsIPv6(_ip.String()) {
		if ip, err := IPv6ToInt(_ip); err != nil {
			return err
		} else {
			numbers = ip[:]
		}
	} else {
		return fmt.Errorf("this is not a valid ip address")
	}
	s, _ := sqids.New(sqids.Options{
		Alphabet: SQID_ALPHABET,
	})
	if _id, err := s.Encode(numbers); err != nil {
		return err
	} else {
		*id = _id
	}
	return nil
}

// conversion credits to: https://github.com/praserx/ipconv/blob/master/ipconv.go
func IPv4ToInt(ipaddr net.IP) (uint32, error) {
	if ipaddr.To4() == nil {
		return 0, fmt.Errorf("this is not an ipv4 address")
	}
	return binary.BigEndian.Uint32(ipaddr.To4()), nil
}

func IPv6ToInt(ipaddr net.IP) ([2]uint64, error) {
	if ipaddr.To16()[0:8] == nil || ipaddr.To16()[8:16] == nil {
		return [2]uint64{0, 0}, fmt.Errorf("this is not an ipv6 address")
	}

	// Get two separates values of integer IP
	ip := [2]uint64{
		binary.BigEndian.Uint64(ipaddr.To16()[0:8]),  // IP high
		binary.BigEndian.Uint64(ipaddr.To16()[8:16]), // IP low
	}

	return ip, nil
}

func IsIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

func IsIPv6(address string) bool {
	return strings.Count(address, ":") >= 2
}
