package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Migration struct {
	Version     int
	Description string
	Up          func(*mongo.Database) error
	Down        func(*mongo.Database) error
}

type Migrator struct {
	db         *mongo.Database
	migrations []Migration
}

func NewMigrator(db *mongo.Database) *Migrator {
	return &Migrator{
		db:         db,
		migrations: getMigrations(),
	}
}

func (m *Migrator) Up() error {
	// Create migrations collection if it doesn't exist
	err := m.createMigrationsCollection()
	if err != nil {
		return err
	}

	// Get current version
	currentVersion, err := m.getCurrentVersion()
	if err != nil {
		return err
	}

	// Run migrations
	for _, migration := range m.migrations {
		if migration.Version > currentVersion {
			log.Printf("Running migration %d: %s", migration.Version, migration.Description)

			err := migration.Up(m.db)
			if err != nil {
				return fmt.Errorf("migration %d failed: %w", migration.Version, err)
			}

			err = m.updateVersion(migration.Version)
			if err != nil {
				return fmt.Errorf("failed to update migration version: %w", err)
			}

			log.Printf("Migration %d completed successfully", migration.Version)
		}
	}

	return nil
}

func (m *Migrator) Down(targetVersion int) error {
	currentVersion, err := m.getCurrentVersion()
	if err != nil {
		return err
	}

	for i := len(m.migrations) - 1; i >= 0; i-- {
		migration := m.migrations[i]
		if migration.Version <= currentVersion && migration.Version > targetVersion {
			log.Printf("Reverting migration %d: %s", migration.Version, migration.Description)

			err := migration.Down(m.db)
			if err != nil {
				return fmt.Errorf("migration %d rollback failed: %w", migration.Version, err)
			}

			previousVersion := targetVersion
			if i > 0 {
				previousVersion = m.migrations[i-1].Version
			}

			err = m.updateVersion(previousVersion)
			if err != nil {
				return fmt.Errorf("failed to update migration version: %w", err)
			}

			log.Printf("Migration %d reverted successfully", migration.Version)
		}
	}

	return nil
}

func (m *Migrator) createMigrationsCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := m.db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return err
	}

	for _, name := range collections {
		if name == "migrations" {
			return nil
		}
	}

	return m.db.CreateCollection(ctx, "migrations")
}

func (m *Migrator) getCurrentVersion() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		Version int `bson:"version"`
	}

	err := m.db.Collection("migrations").FindOne(ctx, bson.D{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}

	return result.Version, nil
}

func (m *Migrator) updateVersion(version int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.db.Collection("migrations").ReplaceOne(
		ctx,
		bson.D{},
		bson.D{{"version", version}, {"updated_at", time.Now()}},
		options.Replace().SetUpsert(true),
	)

	return err
}

func getMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Create users collection with indexes",
			Up: func(db *mongo.Database) error {
				return createUsersIndexes(db)
			},
			Down: func(db *mongo.Database) error {
				return db.Collection("users").Drop(context.Background())
			},
		},
		{
			Version:     2,
			Description: "Create drivers collection with indexes",
			Up: func(db *mongo.Database) error {
				return createDriversIndexes(db)
			},
			Down: func(db *mongo.Database) error {
				return db.Collection("drivers").Drop(context.Background())
			},
		},
		{
			Version:     3,
			Description: "Create riders collection with indexes",
			Up: func(db *mongo.Database) error {
				return createRidersIndexes(db)
			},
			Down: func(db *mongo.Database) error {
				return db.Collection("riders").Drop(context.Background())
			},
		},
		{
			Version:     4,
			Description: "Create rides collection with indexes",
			Up: func(db *mongo.Database) error {
				return createRidesIndexes(db)
			},
			Down: func(db *mongo.Database) error {
				return db.Collection("rides").Drop(context.Background())
			},
		},
		{
			Version:     5,
			Description: "Create vehicles collection with indexes",
			Up: func(db *mongo.Database) error {
				return createVehiclesIndexes(db)
			},
			Down: func(db *mongo.Database) error {
				return db.Collection("vehicles").Drop(context.Background())
			},
		},
		{
			Version:     6,
			Description: "Create payments collection with indexes",
			Up: func(db *mongo.Database) error {
				return createPaymentsIndexes(db)
			},
			Down: func(db *mongo.Database) error {
				return db.Collection("payments").Drop(context.Background())
			},
		},
	}
}

func createUsersIndexes(db *mongo.Database) error {
	ctx := context.Background()
	collection := db.Collection("users")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"email", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{"phone", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"user_type", 1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func createDriversIndexes(db *mongo.Database) error {
	ctx := context.Background()
	collection := db.Collection("drivers")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"user_id", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"current_location", "2dsphere"}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"is_available", 1}},
		},
		{
			Keys: bson.D{{"rating", -1}},
		},
		{
			Keys: bson.D{{"license_number", 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func createRidersIndexes(db *mongo.Database) error {
	ctx := context.Background()
	collection := db.Collection("riders")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"user_id", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"rating", -1}},
		},
		{
			Keys: bson.D{{"total_rides", -1}},
		},
		{
			Keys: bson.D{{"referral_code", 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func createRidesIndexes(db *mongo.Database) error {
	ctx := context.Background()
	collection := db.Collection("rides")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"rider_id", 1}},
		},
		{
			Keys: bson.D{{"driver_id", 1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"pickup_location", "2dsphere"}},
		},
		{
			Keys: bson.D{{"dropoff_location", "2dsphere"}},
		},
		{
			Keys: bson.D{{"requested_at", -1}},
		},
		{
			Keys:    bson.D{{"ride_number", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"scheduled_time", 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func createVehiclesIndexes(db *mongo.Database) error {
	ctx := context.Background()
	collection := db.Collection("vehicles")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"driver_id", 1}},
		},
		{
			Keys:    bson.D{{"license_plate", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{"vin", 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"vehicle_type_id", 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func createPaymentsIndexes(db *mongo.Database) error {
	ctx := context.Background()
	collection := db.Collection("payments")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"ride_id", 1}},
		},
		{
			Keys: bson.D{{"payer_id", 1}},
		},
		{
			Keys: bson.D{{"payee_id", 1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys:    bson.D{{"transaction_id", 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}
