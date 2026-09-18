package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate")

type Repository struct {
	users *mongo.Collection
	polls *mongo.Collection
	votes *mongo.Collection
}

func New(db *mongo.Database) (*Repository, error) {
	r := &Repository{
		users: db.Collection("users"),
		polls: db.Collection("polls"),
		votes: db.Collection("votes"),
	}

	_, err := r.votes.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "pollId", Value: 1},
			{Key: "voterId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	_, err = r.users.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return r, err
}

func (r *Repository) CreateUser(ctx context.Context, email, hash string) (primitive.ObjectID, error) {
	doc := bson.M{
		"email":        email,
		"passwordHash": hash,
		"createdAt":    time.Now(),
	}

	res, err := r.users.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return primitive.NilObjectID, ErrDuplicate
		}
		return primitive.NilObjectID, err
	}

	return res.InsertedID.(primitive.ObjectID), nil
}

func (r *Repository) FindUser(ctx context.Context, email string) (bson.M, error) {
	var u bson.M

	err := r.users.FindOne(ctx, bson.M{"email": email}).Decode(&u)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}

	return u, err
}

func (r *Repository) CreatePoll(
	ctx context.Context,
	owner primitive.ObjectID,
	question string,
	options []string,
) (primitive.ObjectID, error) {

	doc := bson.M{
		"ownerId":   owner,
		"question":  question,
		"options":   options,
		"open":      true,
		"createdAt": time.Now(),
	}

	res, err := r.polls.InsertOne(ctx, doc)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return res.InsertedID.(primitive.ObjectID), nil
}

func (r *Repository) GetPoll(ctx context.Context, id primitive.ObjectID) (bson.M, error) {
	var p bson.M

	err := r.polls.FindOne(ctx, bson.M{"_id": id}).Decode(&p)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}

	return p, err
}

func (r *Repository) ClosePoll(
	ctx context.Context,
	id primitive.ObjectID,
	owner primitive.ObjectID,
) error {

	res, err := r.polls.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"ownerId": owner,
		},
		bson.M{
			"$set": bson.M{"open": false},
		},
	)

	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) CreateVote(
	ctx context.Context,
	pollID primitive.ObjectID,
	option int,
	voterID string,
) error {

	_, err := r.votes.InsertOne(ctx, bson.M{
		"pollId":    pollID,
		"option":    option,
		"voterId":   voterID,
		"createdAt": time.Now(),
	})

	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicate
	}

	return err
}

func (r *Repository) Counts(
	ctx context.Context,
	pollID primitive.ObjectID,
	optionCount int,
) ([]int64, error) {

	counts := make([]int64, optionCount)

	cur, err := r.votes.Aggregate(
		ctx,
		mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"pollId": pollID}}},
			{{Key: "$group", Value: bson.M{
				"_id":   "$option",
				"count": bson.M{"$sum": 1},
			}}},
		},
	)

	if err != nil {
		return counts, err
	}

	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var row struct {
			ID    int   `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cur.Decode(&row); err != nil {
			return counts, err
		}

		if row.ID >= 0 && row.ID < optionCount {
			counts[row.ID] = row.Count
		}
	}

	return counts, cur.Err()
}
