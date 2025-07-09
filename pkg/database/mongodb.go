package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	Config   *DatabaseConfig
}

type DatabaseConfig struct {
	URI            string
	Database       string
	MaxPoolSize    int
	MinPoolSize    int
	ConnectTimeout time.Duration
	SocketTimeout  time.Duration
}

func NewMongoDB(config *DatabaseConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	// Create client options
	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetMaxPoolSize(uint64(config.MaxPoolSize)).
		SetMinPoolSize(uint64(config.MinPoolSize)).
		SetSocketTimeout(config.SocketTimeout).
		SetConnectTimeout(config.ConnectTimeout)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the database
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	database := client.Database(config.Database)

	return &MongoDB{
		Client:   client,
		Database: database,
		Config:   config,
	}, nil
}

func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

func (m *MongoDB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Ping(ctx, readpref.Primary())
}

func (m *MongoDB) CreateTransaction() (mongo.Session, error) {
	return m.Client.StartSession()
}

func (m *MongoDB) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) (interface{}, error)) (interface{}, error) {
	session, err := m.CreateTransaction()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	return session.WithTransaction(ctx, fn)
}
