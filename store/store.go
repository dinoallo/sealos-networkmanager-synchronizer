package store

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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

func (s *Store) IncBytesField(ctx context.Context, nn string, addr string, tag string, bytes uint64, t int) error {
	log := s.Log
	if log == nil {
		return nil
	}
	var bytesFieldName string
	if t == 0 {
		bytesFieldName = "recv_bytes"
	} else if t == 1 {
		bytesFieldName = "sent_bytes"
	} else {
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
		Value: nn,
	}}
	key := fmt.Sprintf("address_properties.%s.tag_properties.%s.%s", addr, tag, bytesFieldName)
	update := bson.D{{
		Key: "$inc",
		Value: bson.D{{
			Key:   key,
			Value: bytes,
		}},
	}}
	if _, err := coll.UpdateOne(updateCtx, filter, update, opts); err != nil {
		return err
	} else {
		log.Info("the data has been updated", "field", key)
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
