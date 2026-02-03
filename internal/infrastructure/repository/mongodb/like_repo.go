package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ErrReactionNotFound is returned when a reaction is not found in the database.
var ErrReactionNotFound = errors.New("reaction not found")

// LikeRepository represents the MongoDB implementation of the ILikeRepository interface.
type LikeRepository struct {
	collection *mongo.Collection
}

// NewLikeRepository creates and returns a new LikeRepository instance.
func NewLikeRepository(db *mongo.Database) *LikeRepository {
	return &LikeRepository{
		collection: db.Collection("blog_likes"),
	}
}

// CreateReaction creates or updates a user's reaction (like/dislike) on a target.
func (r *LikeRepository) CreateReaction(ctx context.Context, like *entity.Like) error {
	// Filter to find an existing reaction by this user on this target.
	filter := bson.M{
		"user_id":     like.UserID,
		"target_id":   like.TargetID,
		"target_type": like.TargetType,
	}

	// Fields to set/update on the document.
	updateFields := bson.M{
		"type":       like.Type,
		"is_deleted": false,
		"updated_at": time.Now(),
	}

	// Fields to set ONLY on initial insert (when upsert: true creates a new document)
	setOnInsertFields := bson.M{
		"_id":        uuid.New().String(),
		"created_at": time.Now(),
	}

	updateDoc := bson.M{
		"$set":         updateFields,
		"$setOnInsert": setOnInsertFields,
	}

	opts := options.Update().SetUpsert(true)

	res, err := r.collection.UpdateOne(ctx, filter, updateDoc, opts)
	if err != nil {
		return fmt.Errorf("failed to create or update reaction record: %w", err)
	}

	if res.UpsertedID != nil {
		if id, ok := res.UpsertedID.(string); ok {
			like.ID = id
		} else {
			return fmt.Errorf("upserted ID is not a string, got type %T", res.UpsertedID)
		}
		like.CreatedAt = setOnInsertFields["created_at"].(time.Time)
	}

	return nil
}

// DeleteReaction marks a reaction record as deleted (soft delete) by its unique ID.
func (r *LikeRepository) DeleteReaction(ctx context.Context, reactionID string) error {
	filter := bson.M{"_id": reactionID, "is_deleted": false}
	update := bson.M{"$set": bson.M{"is_deleted": true, "updated_at": time.Now()}}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}
	if res.ModifiedCount == 0 {
		return ErrReactionNotFound
	}
	return nil
}

// GetReactionByUserIDAndTargetID retrieves any active reaction (like or dislike) by a specific user on a specific target.
func (r *LikeRepository) GetReactionByUserIDAndTargetID(ctx context.Context, userID, targetID string) (*entity.Like, error) {
	var like entity.Like
	// Filter for active reactions
	filter := bson.M{"user_id": userID, "target_id": targetID, "is_deleted": false}

	err := r.collection.FindOne(ctx, filter).Decode(&like)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrReactionNotFound
		}
		return nil, fmt.Errorf("failed to retrieve reaction: %w", err)
	}
	return &like, nil
}

// GetReactionByUserIDTargetIDAndType retrieves a specific type of active reaction (like or dislike) by a user on a target.
func (r *LikeRepository) GetReactionByUserIDTargetIDAndType(ctx context.Context, userID, targetID string, reactionType entity.LikeType) (*entity.Like, error) {
	var like entity.Like
	filter := bson.M{
		"user_id":    userID,
		"target_id":  targetID,
		"type":       reactionType,
		"is_deleted": false,
	}

	err := r.collection.FindOne(ctx, filter).Decode(&like)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrReactionNotFound
		}
		return nil, fmt.Errorf("failed to retrieve specific reaction: %w", err)
	}
	return &like, nil
}

// CountLikesByTargetID counts the number of active 'like' reactions for a specific target.
func (r *LikeRepository) CountLikesByTargetID(ctx context.Context, targetID string) (int64, error) {
	// Filter to count only active 'likes'
	filter := bson.M{"target_id": targetID, "type": entity.LIKE_TYPE_LIKE, "is_deleted": false}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count active likes: %w", err)
	}
	return count, nil
}

// CountDislikesByTargetID counts the number of active 'dislike' reactions for a specific target.
func (r *LikeRepository) CountDislikesByTargetID(ctx context.Context, targetID string) (int64, error) {
	// Filter to count only active 'dislikes'
	filter := bson.M{"target_id": targetID, "type": entity.LIKE_TYPE_DISLIKE, "is_deleted": false}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count active dislikes: %w", err)
	}
	return count, nil
}
