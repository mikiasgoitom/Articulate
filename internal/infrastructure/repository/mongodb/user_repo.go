package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoUserRepository struct {
	collection *mongo.Collection
}

func NewMongoUserRepository(collection *mongo.Collection) *MongoUserRepository {
	return &MongoUserRepository{collection: collection}
}

func (r *MongoUserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *MongoUserRepository) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (r *MongoUserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) GetByUserName(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

// UpdateUser updates an existing user and returns the updated user
func (r *MongoUserRepository) UpdateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	user.UpdatedAt = time.Now()
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": user}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("UpdateOne error: %v", err)
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, errors.New("user not found")
	}
	var updatedUser entity.User
	if err := r.collection.FindOne(ctx, filter).Decode(&updatedUser); err != nil {
		return nil, err
	}
	return &updatedUser, nil
}

func (r *MongoUserRepository) UpdateUserPassword(ctx context.Context, id string, hashedPassword string) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"password_hash": hashedPassword}}
	count, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if count.MatchedCount == 0 {
		return fmt.Errorf("failed to fetch user with id:%s", id)
	}
	return nil
}

func (r *MongoUserRepository) DeleteUser(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}
	count, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if count.DeletedCount == 0 {
		return fmt.Errorf("failed to fetch user with id:%s", id)
	}
	return nil
}
