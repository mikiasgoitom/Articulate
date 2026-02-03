package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MediaRepository represents the MongoDB implementation of the IMediaRepository interface.
type MediaRepository struct {
	collection *mongo.Collection
}

// NewMediaRepository creates and returns a new MediaRepository instance.
func NewMediaRepository(db *mongo.Database) *MediaRepository {
	return &MediaRepository{
		collection: db.Collection("media"),
	}
}

// CreateMedia inserts a new media record into the database.
func (r *MediaRepository) CreateMedia(ctx context.Context, media *entity.Media) error {
	media.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, media)
	if err != nil {
		return fmt.Errorf("failed to create media record: %w", err)
	}
	return nil
}

// GetMediaByID retrieves a single media record by its unique ID, excluding soft-deleted records.
func (r *MediaRepository) GetMediaByID(ctx context.Context, mediaID string) (*entity.Media, error) {
	var media entity.Media
	filter := bson.M{
		"_id":        mediaID,
		"is_deleted": false,
	}

	err := r.collection.FindOne(ctx, filter).Decode(&media)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("media with ID %s not found or has been deleted: %w", mediaID, err)
		}
		return nil, fmt.Errorf("failed to retrieve media record with ID %s: %w", mediaID, err)
	}
	return &media, nil
}

// GetMediaParams holds parameters for filtering, sorting, and pagination.
type GetMediaParams struct {
	Filter bson.M
	Sort   bson.M
	Limit  int64
	Skip   int64
}

// GetMedia retrieves a list of media records based on the provided parameters, excluding soft-deleted records.
func (r *MediaRepository) GetMedia(ctx context.Context, params GetMediaParams) ([]*entity.Media, error) {
	baseFilter := bson.M{"is_deleted": false}
	if params.Filter != nil {
		for key, value := range params.Filter {
			baseFilter[key] = value
		}
	}

	opts := options.Find()
	if params.Limit > 0 {
		opts.SetLimit(params.Limit)
	}
	if params.Skip > 0 {
		opts.SetSkip(params.Skip)
	}
	if params.Sort != nil {
		opts.SetSort(params.Sort)
	}

	cursor, err := r.collection.Find(ctx, baseFilter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve media records: %w", err)
	}
	defer cursor.Close(ctx)

	var mediaList []*entity.Media
	if err = cursor.All(ctx, &mediaList); err != nil {
		return nil, fmt.Errorf("failed to decode media records: %w", err)
	}

	if len(mediaList) == 0 {
		return []*entity.Media{}, nil
	}

	return mediaList, nil
}

// UpdateMedia updates an existing media record by its ID.
func (r *MediaRepository) UpdateMedia(ctx context.Context, mediaID string, updates bson.M) error {
	filter := bson.M{
		"_id":        mediaID,
		"is_deleted": false,
	}

	update := bson.M{
		"$set": updates,
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update media record with ID %s: %w", mediaID, err)
	}

	if res.ModifiedCount == 0 {
		var media entity.Media
		err := r.collection.FindOne(ctx, bson.M{"_id": mediaID}).Decode(&media)
		if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("media record with ID %s not found", mediaID)
		}
		return fmt.Errorf("media record with ID %s was not modified (no new data to apply)", mediaID)
	}

	return nil
}

// DeleteMedia soft deletes a media record by its ID.
func (r *MediaRepository) DeleteMedia(ctx context.Context, mediaID string) error {
	filter := bson.M{"_id": mediaID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to soft-delete media record with ID %s: %w", mediaID, err)
	}

	if res.ModifiedCount == 0 {
		var media entity.Media
		err := r.collection.FindOne(ctx, bson.M{"_id": mediaID}).Decode(&media)
		if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("media record with ID %s not found", mediaID)
		}
		return fmt.Errorf("media record with ID %s was not modified (possibly already deleted)", mediaID)
	}

	return nil
}

// AssociateMediaWithBlog sets the BlogID for a media record.
func (r *MediaRepository) AssociateMediaWithBlog(ctx context.Context, mediaID, blogID string) error {
	filter := bson.M{"_id": mediaID, "is_deleted": false}
	update := bson.M{"$set": bson.M{"blog_id": blogID}}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to associate media %s with blog %s: %w", mediaID, blogID, err)
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("media record with ID %s not found or already associated", mediaID)
	}
	return nil
}

func (r *MediaRepository) RemoveMediaFromBlog(ctx context.Context, mediaID string) error {
	filter := bson.M{"_id": mediaID, "is_deleted": false}
	update := bson.M{"$unset": bson.M{"blog_id": ""}}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to remove blog association from media %s: %w", mediaID, err)
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("media record with ID %s not found or not associated with a blog", mediaID)
	}
	return nil
}

// GetMediaByBlogID retrieves all media associated with a specific blog.
func (r *MediaRepository) GetMediaByBlogID(ctx context.Context, blogID string) ([]*entity.Media, error) {
	filter := bson.M{"blog_id": blogID, "is_deleted": false}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve media for blog %s: %w", blogID, err)
	}
	defer cursor.Close(ctx)

	var mediaList []*entity.Media
	if err = cursor.All(ctx, &mediaList); err != nil {
		return nil, fmt.Errorf("failed to decode media records: %w", err)
	}
	return mediaList, nil
}
