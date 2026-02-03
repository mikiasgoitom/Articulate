package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/uuidgen"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrCommentNotFound     = errors.New("comment not found")
	ErrCommentCreation     = errors.New("failed to create comment")
	ErrCommentUpdate       = errors.New("failed to update comment")
	ErrCommentDeletion     = errors.New("failed to delete comment")
	ErrInvalidPagination   = errors.New("invalid pagination parameters")
	ErrInvalidParentTarget = errors.New("invalid parent/target relationship")
	ErrCommentAlreadyLiked = errors.New("comment already liked by user")
	ErrCommentNotLiked     = errors.New("comment not liked by user")
)

type CommentRepository struct {
	collection       *mongo.Collection
	likeCollection   *mongo.Collection
	reportCollection *mongo.Collection
}

func NewCommentRepository(db *mongo.Database) *CommentRepository {
	return &CommentRepository{
		collection:       db.Collection("comments"),
		likeCollection:   db.Collection("comment_likes"),
		reportCollection: db.Collection("comment_reports"),
	}
}

// Pagination struct removed; use contract.Pagination instead.

// Core CRUD Operations
func (r *CommentRepository) Create(ctx context.Context, comment *entity.Comment) error {
	comment.ID = uuidgen.NewGenerator().NewUUID()

	// Validate parent/target logic
	if err := r.validateParentTargetLogic(ctx, comment); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParentTarget, err)
	}

	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	comment.IsDeleted = false

	if comment.Status == "" {
		comment.Status = "approved"
	}

	_, err := r.collection.InsertOne(ctx, comment)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommentCreation, err)
	}

	return nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*entity.Comment, error) {
	var comment entity.Comment
	filter := bson.M{"_id": id, "is_deleted": false}

	err := r.collection.FindOne(ctx, filter).Decode(&comment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return &comment, nil
}

func (r *CommentRepository) Update(ctx context.Context, comment *entity.Comment) error {
	comment.UpdatedAt = time.Now()

	filter := bson.M{"_id": comment.ID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"content":    comment.Content,
			"updated_at": comment.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommentUpdate, err)
	}

	if result.MatchedCount == 0 {
		return ErrCommentNotFound
	}

	return nil
}

func (r *CommentRepository) Delete(ctx context.Context, id string) error {
	filter := bson.M{"_id": id, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommentDeletion, err)
	}

	if result.MatchedCount == 0 {
		return ErrCommentNotFound
	}

	return nil
}

// Listing Operations
func (r *CommentRepository) GetTopLevelComments(ctx context.Context, blogID string, pagination contract.Pagination) (comments []*entity.Comment, total int64, err error) {
	if pagination.Page < 1 || pagination.PageSize < 1 {
		return nil, 0, ErrInvalidPagination
	}

	filter := bson.M{
		"blog_id":    blogID,
		"parent_id":  nil,
		"is_deleted": false,
		"status":     bson.M{"$in": []string{"approved"}},
	}

	// Get total count
	total, err = r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments: %w", err)
	}

	// Get paginated results
	skip := int64((pagination.Page - 1) * pagination.PageSize)
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(pagination.PageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find comments: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, fmt.Errorf("failed to decode comments: %w", err)
	}

	return comments, total, nil
}

func (r *CommentRepository) GetCommentThread(ctx context.Context, parentID string) (*entity.CommentThread, error) {
	// Get the parent comment
	parentComment, err := r.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	// If this is not a top-level comment, return error
	if parentComment.ParentID != nil {
		return nil, errors.New("can only get thread for top-level comments")
	}

	thread := &entity.CommentThread{
		Comment: parentComment,
		Replies: []*entity.CommentThread{},
		Depth:   0,
	}

	// Get all replies for this thread
	replies, err := r.getRepliesRecursively(ctx, parentID, 1)
	if err != nil {
		return nil, err
	}

	thread.Replies = replies
	return thread, nil
}

func (r *CommentRepository) GetCommentsByUser(ctx context.Context, userID string, pagination contract.Pagination) ([]*entity.Comment, int64, error) {
	if pagination.Page < 1 || pagination.PageSize < 1 {
		return nil, 0, ErrInvalidPagination
	}

	filter := bson.M{
		"author_id":  userID,
		"is_deleted": false,
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user comments: %w", err)
	}

	skip := int64((pagination.Page - 1) * pagination.PageSize)
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(pagination.PageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find user comments: %w", err)
	}
	defer cursor.Close(ctx)

	var comments []*entity.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, fmt.Errorf("failed to decode user comments: %w", err)
	}

	return comments, total, nil
}

// Status and Moderation
func (r *CommentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	filter := bson.M{"_id": id, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update comment status: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrCommentNotFound
	}

	return nil
}

func (r *CommentRepository) GetCommentCount(ctx context.Context, blogID string) (int64, error) {
	filter := bson.M{
		"blog_id":    blogID,
		"is_deleted": false,
		"status":     "approved",
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return count, nil
}

// Like System
func (r *CommentRepository) LikeComment(ctx context.Context, commentID string, userID string) error {
	// Check if already liked
	exists, err := r.IsCommentLikedByUser(ctx, commentID, userID)
	if err != nil {
		return err
	}
	if exists {
		return ErrCommentAlreadyLiked
	}

	// Create like record
	like := &entity.CommentLike{
		ID:        uuidgen.NewGenerator().NewUUID(),
		CommentID: commentID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Insert like
		_, err := r.likeCollection.InsertOne(sc, like)
		if err != nil {
			return err
		}

		// Increment like count
		filter := bson.M{"_id": commentID}
		update := bson.M{"$inc": bson.M{"like_count": 1}}
		_, err = r.collection.UpdateOne(sc, filter, update)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to like comment: %w", err)
	}

	return nil
}

func (r *CommentRepository) UnlikeComment(ctx context.Context, commentID, userID string) error {
	// Check if liked
	exists, err := r.IsCommentLikedByUser(ctx, commentID, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCommentNotLiked
	}

	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Remove like
		filter := bson.M{"comment_id": commentID, "user_id": userID}
		_, err := r.likeCollection.DeleteOne(sc, filter)
		if err != nil {
			return err
		}

		// Decrement like count
		commentFilter := bson.M{"_id": commentID}
		update := bson.M{"$inc": bson.M{"like_count": -1}}
		_, err = r.collection.UpdateOne(sc, commentFilter, update)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to unlike comment: %w", err)
	}

	return nil
}

func (r *CommentRepository) IsCommentLikedByUser(ctx context.Context, commentID, userID string) (bool, error) {
	filter := bson.M{"comment_id": commentID, "user_id": userID}
	count, err := r.likeCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check like status: %w", err)
	}
	return count > 0, nil
}

func (r *CommentRepository) GetCommentLikeCount(ctx context.Context, commentID string) (int64, error) {
	filter := bson.M{"comment_id": commentID}
	count, err := r.likeCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to get like count: %w", err)
	}
	return count, nil
}

// Reporting System
func (r *CommentRepository) ReportComment(ctx context.Context, report *entity.CommentReport) error {
	if report.ID == "" {
		report.ID = uuidgen.NewGenerator().NewUUID()
	}
	report.CreatedAt = time.Now()
	report.Status = "pending"

	_, err := r.reportCollection.InsertOne(ctx, report)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

func (r *CommentRepository) GetCommentReports(ctx context.Context, pagination contract.Pagination) ([]*entity.CommentReport, int64, error) {
	if pagination.Page < 1 || pagination.PageSize < 1 {
		return nil, 0, ErrInvalidPagination
	}

	filter := bson.M{}

	total, err := r.reportCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reports: %w", err)
	}

	skip := int64((pagination.Page - 1) * pagination.PageSize)
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(pagination.PageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.reportCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find reports: %w", err)
	}
	defer cursor.Close(ctx)

	var reports []*entity.CommentReport
	if err := cursor.All(ctx, &reports); err != nil {
		return nil, 0, fmt.Errorf("failed to decode reports: %w", err)
	}

	return reports, total, nil
}

func (r *CommentRepository) UpdateReportStatus(ctx context.Context, reportID string, status string, reviewerID string) error {
	filter := bson.M{"_id": reportID}
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":      status,
			"reviewed_at": &now,
			"reviewed_by": &reviewerID,
		},
	}

	result, err := r.reportCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update report status: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("report not found")
	}

	return nil
}

// Helper Methods
func (r *CommentRepository) validateParentTargetLogic(ctx context.Context, comment *entity.Comment) error {
	// If no parent, this is a top-level comment
	if comment.ParentID == nil {
		if comment.TargetID != nil {
			return errors.New("top-level comments cannot have target_id")
		}
		return nil
	}

	// Validate parent exists and is top-level
	parent, err := r.GetByID(ctx, *comment.ParentID)
	if err != nil {
		return fmt.Errorf("parent comment not found: %w", err)
	}

	if parent.ParentID != nil {
		return errors.New("parent must be a top-level comment")
	}

	// Validate target if specified
	if comment.TargetID != nil {
		target, err := r.GetByID(ctx, *comment.TargetID)
		if err != nil {
			return fmt.Errorf("target comment not found: %w", err)
		}

		// Target must be either the parent or a reply in the same thread
		if target.ID != *comment.ParentID &&
			(target.ParentID == nil || *target.ParentID != *comment.ParentID) {
			return errors.New("target comment must be in the same thread")
		}
	}

	return nil
}

func (r *CommentRepository) getRepliesRecursively(ctx context.Context, parentID string, depth int) ([]*entity.CommentThread, error) {
	if depth > contract.MaxCommentDepth { // Prevent excessive nesting
		return []*entity.CommentThread{}, nil
	}

	filter := bson.M{
		"parent_id":  parentID,
		"is_deleted": false,
		"status":     "approved",
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find replies: %w", err)
	}
	defer cursor.Close(ctx)

	var replies []*entity.Comment
	if err := cursor.All(ctx, &replies); err != nil {
		return nil, fmt.Errorf("failed to decode replies: %w", err)
	}

	var threads []*entity.CommentThread
	for _, reply := range replies {
		thread := &entity.CommentThread{
			Comment: reply,
			Depth:   depth,
		}

		// Get nested replies
		nestedReplies, err := r.getRepliesRecursively(ctx, reply.ID, depth+1)
		if err != nil {
			return nil, err
		}
		thread.Replies = nestedReplies

		threads = append(threads, thread)
	}

	return threads, nil
}
