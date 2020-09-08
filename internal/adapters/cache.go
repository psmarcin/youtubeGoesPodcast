package adapters

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"github.com/psmarcin/youtubegoespodcast/internal/config"
	"github.com/sirupsen/logrus"
	"time"
)

// Cache keeps firestore client and collection
type Cache struct {
	firestore  *firestore.Client
	collection *firestore.CollectionRef
}

// CacheEntity is a model for Firestore that handles cache with ttl
type CacheEntity struct {
	Key   string        `firestore:"key"`
	Value string        `firestore:"value"`
	Ttl   time.Duration `firestore:"ttl"`
}

var l = logrus.WithField("source", "adapter")

// NewCacheRepository establishes connection to Firebase and update singleton variable
func NewCacheRepository() (Cache, error) {
	cache := Cache{}
	ctx := context.Background()
	store, err := firestore.NewClient(ctx, firestore.DetectProjectID)
	if err != nil {
		logrus.WithError(err).Errorf("can't connect to Firebase")
		return cache, err
	}
	//defer store.Close() // Close client when done.

	cache.firestore = store
	cache.collection = store.Collection(config.Cfg.FirestoreCollection)

	return cache, nil
}

// SetKey saves value for key in cache store with expiration time
func (c *Cache) SetKey(key, value string, exp time.Duration) error {
	// todo: pass external context
	ctx := context.Background()

	_, err := c.collection.Doc(key).Set(ctx, CacheEntity{
		Key:   key,
		Value: value,
		Ttl:   exp,
	})
	if err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"key":   key,
			"value": value,
			"exp":   exp.String(),
		}).Errorf("set failed for %s", key)
		return err
	}

	return nil
}

// GetKey retrieve value by key from cache store
func (c *Cache) GetKey(key string) (string, error) {
	// todo: pass external context
	ctx := context.Background()
	raw, err := c.collection.Doc(key).Get(ctx)
	if err != nil {
		l.WithError(err).Warnf("can't get document %s", key)
		return "", err
	}

	var e CacheEntity
	err = raw.DataTo(&e)
	if err != nil {
		l.WithError(err).Errorf("can't parse document %s", key)
		return "", err
	}

	timeDiff := time.Now().Sub(raw.UpdateTime)
	if timeDiff > e.Ttl {
		l.Debugf("key expires, updated at %s, expires in %s", raw.UpdateTime.Format(time.RFC3339), e.Ttl.String())
		return "", errors.New("Cache expires for " + key)
	}

	l.Debugf("got key %s", key)

	return e.Value, nil
}
