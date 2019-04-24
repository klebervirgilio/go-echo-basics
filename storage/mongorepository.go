package mongorepository

import (
	"log"

	"github.com/klebervirgilio/go-echo-basics/config"
	"github.com/klebervirgilio/go-echo-basics/core"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func NewMongoRepo(config *config.Config) core.Repository {
	client := newMongoClient(
		config.GetString("mongo.uri"),
		config.GetString("mongo.database.name"),
		config.GetString("mongo.collection.name"),
	)
	return MongoRepo{client}
}

// MongoRepo its a concrete implementation of SubscriptionRepo.
type MongoRepo struct {
	client MongoClient
}

func (m MongoRepo) FindAll(selector map[string]interface{}) ([]core.Subscription, error) {
	coll, cs := m.client.GetSession()
	defer cs()

	var subscriptions []core.Subscription
	err := coll.Find(selector).All(&subscriptions)

	return subscriptions, err
}

func (m MongoRepo) Remove(selector map[string]interface{}) error {
	coll, cs := m.client.GetSession()
	defer cs()

	return coll.Remove(selector)
}

func (m MongoRepo) Upsert(subscription core.Subscription) error {
	coll, cs := m.client.GetSession()
	defer cs()
	_, err := coll.Upsert(bson.M{"email": subscription.Email}, subscription)
	return err
}

// MongoClient wraps the mgo package.
type MongoClient struct {
	databaseName   string
	collectionName string
	session        *mgo.Session
}

func newMongoClient(uri, database, collection string) MongoClient {
	mongo, err := mgo.Dial(uri)
	if err != nil {
		log.Fatal(err)
	}
	return MongoClient{
		session:        mongo,
		databaseName:   database,
		collectionName: collection,
	}
}

func (m MongoClient) GetSession() (*mgo.Collection, func()) {
	s := m.session.Copy()
	return s.DB(m.databaseName).C(m.collectionName), s.Close
}
