package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TagRepository represents the MongoDB implementation of the ITagRepository interface.
type TagRepository struct {
	collection *mongo.Collection
}

// NewTagRepository creates and returns a new TagRepository instance.
func NewTagRepository(db *mongo.Database) *TagRepository {
	return &TagRepository{
		collection: db.Collection("tags"),
	}
}

// CreateTag inserts a new tag record into the database.
func (r *TagRepository) CreateTag(ctx context.Context, tag *entity.Tag) error {
	tag.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, tag)
	if err != nil {
		var writeException mongo.WriteException
		if errors.As(err, &writeException) {
			for _, e := range writeException.WriteErrors {
				if e.Code == 11000 {
					return errors.New("tag with this name or slug already exists")
				}
			}
		}
		return fmt.Errorf("failed to create tag: %w", err)
	}
	return nil
}

// GetTagByID retrieves a single tag by its unique ID.
func (r *TagRepository) GetTagByID(ctx context.Context, tagID string) (*entity.Tag, error) {
	var tag entity.Tag
	filter := bson.M{"_id": tagID}

	err := r.collection.FindOne(ctx, filter).Decode(&tag)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to retrieve tag: %w", err)
	}
	return &tag, nil
}

// GetTagByName retrieves a single tag by its name.
func (r *TagRepository) GetTagByName(ctx context.Context, name string) (*entity.Tag, error) {
	var tag entity.Tag
	filter := bson.M{"name": name}

	err := r.collection.FindOne(ctx, filter).Decode(&tag)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to retrieve tag by name: %w", err)
	}
	return &tag, nil
}

// GetAllTags retrieves all tag records from the database.
func (r *TagRepository) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tags: %w", err)
	}
	defer cursor.Close(ctx)

	var tags []*entity.Tag
	if err = cursor.All(ctx, &tags); err != nil {
		return nil, fmt.Errorf("failed to decode tags: %w", err)
	}
	return tags, nil
}

// UpdateTag updates the details of an existing tag by its ID.
func (r *TagRepository) UpdateTag(ctx context.Context, tagID string, updates map[string]interface{}) error {
	filter := bson.M{"_id": tagID}
	update := bson.M{"$set": updates}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}
	if res.ModifiedCount == 0 {
		return errors.New("tag not found or no changes made")
	}
	return nil
}

// DeleteTag deletes a tag record by its ID.
func (r *TagRepository) DeleteTag(ctx context.Context, tagID string) error {
	filter := bson.M{"_id": tagID}
	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	if res.DeletedCount == 0 {
		return errors.New("tag not found")
	}
	return nil
}
