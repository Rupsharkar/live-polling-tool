package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

type Poll struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID   primitive.ObjectID `bson:"ownerId" json:"ownerId"`
	Question  string             `bson:"question" json:"question"`
	Options   []string           `bson:"options" json:"options"`
	Open      bool               `bson:"open" json:"open"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    primitive.ObjectID `bson:"pollId" json:"pollId"`
	Option    int                `bson:"option" json:"option"`
	VoterID   string             `bson:"voterId" json:"voterId"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
