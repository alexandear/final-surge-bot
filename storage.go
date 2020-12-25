package main

import (
	"encoding/json"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

const (
	BucketUserToken = "UserToken"
)

type Bolt struct {
	db *bolt.DB
}

type UserToken struct {
	UserKey string
	Token   string
}

func NewBolt(db *bolt.DB) (*Bolt, error) {
	b := &Bolt{
		db: db,
	}

	if err := b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(BucketUserToken))
		if errors.Is(err, bolt.ErrBucketExists) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to update: %w", err)
	}

	return b, nil
}

func (b *Bolt) UserToken(userName string) (UserToken, error) {
	var userToken UserToken

	if err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketUserToken))
		bc := b.Get([]byte(userName))
		if bc == nil {
			return nil
		}

		if err := json.Unmarshal(bc, &userToken); err != nil {
			return fmt.Errorf("failed to unmarshal usertoken: %w", err)
		}

		return nil
	}); err != nil {
		return UserToken{}, fmt.Errorf("failed to get user token: %w", err)
	}

	return userToken, nil
}

func (b *Bolt) UpdateUserToken(userName string, userToken UserToken) error {
	bu, err := json.Marshal(userToken)
	if err != nil {
		return fmt.Errorf("failed to marshal user token: %w", err)
	}

	if err := b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketUserToken))

		return b.Put([]byte(userName), bu)
	}); err != nil {
		return fmt.Errorf("failed to put user token: %w", err)
	}

	return nil
}
