package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"latihan-fiber/app/model"
)

var (
	ErrProjectLimit = errors.New("jumlah proyek sudah mencapai batas")
)

const MaxProjects = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	GetByStudentID(ctx context.Context, studentID int) (model.Portfolio, error)
	Upsert(ctx context.Context, studentID int, req model.PutPortfolioRequest) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error)
	DeleteProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error)
	StatsTechnologies(ctx context.Context) ([]model.TechStat, error)
}

type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	Name   string `bson:"name"`
	Issuer string `bson:"issuer"`
	Date   string `bson:"date"`
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		projects = append(projects, model.Project{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: p.Technologies,
			Link:         p.Link,
			Year:         p.Year,
		})
	}
	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{Name: c.Name, Issuer: c.Issuer, Date: c.Date})
	}
	return model.Portfolio{
		ID:           d.ID.Hex(),
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certs,
	}
}

type portfolioMongoRepository struct{ coll *mongo.Collection }

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "student_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("student_id_unique"),
	})
	if err != nil {
		return fmt.Errorf("membuat index portofolio: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) GetByStudentID(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	if err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

func (r *portfolioMongoRepository) Upsert(ctx context.Context, studentID int, req model.PutPortfolioRequest) (model.Portfolio, error) {
	certs := make([]certificateDocument, 0, len(req.Certificates))
	for _, c := range req.Certificates {
		certs = append(certs, certificateDocument{Name: c.Name, Issuer: c.Issuer, Date: c.Date})
	}

	update := bson.M{
		"$set": bson.M{
			"headline":     req.Headline,
			"bio":          req.Bio,
			"certificates": certs,
		},
		"$setOnInsert": bson.M{
			"projects": []projectDocument{}, // set on creation only
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updated portfolioDocument
	if err := r.coll.FindOneAndUpdate(ctx, bson.M{"student_id": studentID}, update, opts).Decode(&updated); err != nil {
		return model.Portfolio{}, fmt.Errorf("upsert portofolio: %w", err)
	}
	return updated.toModel(), nil
}

func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error) {
	doc := projectDocument{
		ID:           bson.NewObjectID(),
		Title:        p.Title,
		Technologies: p.Technologies,
		Link:         p.Link,
		Year:         p.Year,
	}
	limitField := "projects." + strconv.Itoa(MaxProjects-1)
	var updated portfolioDocument

	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, limitField: bson.M{"$exists": false}},
		bson.M{"$push": bson.M{"projects": doc}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)

	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectLimit
}

func (r *portfolioMongoRepository) DeleteProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error) {
	oid, err := parseID(projectID)
	if err != nil {
		return model.Portfolio{}, err
	}
	var updated portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID},
		bson.M{"$pull": bson.M{"projects": bson.M{"_id": oid}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrNotFound
		}
		return model.Portfolio{}, fmt.Errorf("menghapus proyek: %w", err)
	}
	return updated.toModel(), nil
}

func (r *portfolioMongoRepository) StatsTechnologies(ctx context.Context) ([]model.TechStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{"_id": "$projects.technologies", "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: 10}},
	}
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("stats tech: %w", err)
	}
	defer cursor.Close(ctx)

	var out []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("membaca stats: %w", err)
	}

	stats := make([]model.TechStat, 0, len(out))
	for _, o := range out {
		stats = append(stats, model.TechStat{Technology: o.ID, Count: o.Count})
	}
	return stats, nil
}