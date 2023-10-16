package db

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/openshift/sippy/pkg/db/models"
)

type SchemaHashType string

const (
	hashTypeMatView      SchemaHashType = "matview"
	hashTypeMatViewIndex SchemaHashType = "matview_index"
	hashTypeFunction     SchemaHashType = "function"
)

type DB struct {
	DB *gorm.DB

	// BatchSize is used for how many insertions we should do at once. Postgres supports
	// a maximum of 2^16 records per insert.
	BatchSize int
}

// log2LogrusWriter bridges gorm logging to logrus logging.
// All messages will come through at DEBUG level.
type log2LogrusWriter struct {
	entry *log.Entry
}

func (w log2LogrusWriter) Printf(msg string, args ...interface{}) {
	w.entry.Debugf(msg, args...)
}

func New(dsn string, logLevel gormlogger.LogLevel) (*DB, error) {
	gormLogger := gormlogger.New(
		log2LogrusWriter{entry: log.WithField("source", "gorm")},
		gormlogger.Config{
			SlowThreshold:             2 * time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}
	return &DB{
		DB:        db,
		BatchSize: 1024,
	}, nil
}

func (d *DB) UpdateSchema(reportEnd *time.Time) error {

	if err := d.DB.AutoMigrate(&models.ReleaseTag{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ReleasePullRequest{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ReleaseRepository{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ReleaseJobRun{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ProwJob{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ProwJobRun{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.Test{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.Suite{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ProwJobRunTest{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ProwJobRunTestOutput{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ProwJobRunTestOutputMetadata{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.APISnapshot{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.Bug{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.ProwPullRequest{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.SchemaHash{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.PullRequestComment{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.JiraIncident{}); err != nil {
		return err
	}

	if err := d.DB.AutoMigrate(&models.Migration{}); err != nil {
		return err
	}

	// TODO: in the future, we should add an implied migration. If we see a new suite needs to be created,
	// scan all test names for any starting with that prefix, and if found merge all records into a new or modified test
	// with the prefix stripped. This is not necessary today, but in future as new suites are added, there'll be a good
	// change this happens without thinking to update sippy.
	if err := populateTestSuitesInDB(d.DB); err != nil {
		return err
	}

	if err := syncPostgresMaterializedViews(d.DB, reportEnd); err != nil {
		return err
	}

	if err := syncPostgresFunctions(d.DB); err != nil {
		return err
	}

	log.Infof("applying schema migrations...")
	var keys []string
	for k, _ := range migrations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var foundMigration models.Migration
		if err := d.DB.Model(&models.Migration{}).Where("name = ?", k).First(&foundMigration).Error; err == nil {
			log.Debugf("skipping already applied migration %q", k)
			continue
		} else if err != gorm.ErrRecordNotFound {
			log.WithError(err).Warningf("encountered error while trying to find migration...")
			return err
		}

		log.Infof("applying migration %q...", k)
		err := d.DB.Transaction(func(tx *gorm.DB) error {
			if err := migrations[k](tx); err != nil {
				return err
			}
			d.DB.Save(&models.Migration{
				Name: k,
			})
			return nil
		})
		if err != nil {
			log.WithError(err).Warningf("error applying migration %q", err)
			return err
		}
		log.Infof("migration %q applied", k)
	}
	log.Info("migrations complete")
	log.Info("db schema updated")

	return nil
}

// syncSchema will update generic db resources if their schema has changed. (functions, materialized views, indexes)
// This is useful for resources that cannot be updated incrementally with goose, and can cause conflict / last write
// wins problems with concurrent development.
//
// desiredSchema should be the full SQL command we would issue to create the resource fresh. It will be hashed and
// compared to a pre-existing value in the db of the given name and type, if any exists. If none exists, or the hashes
// have changed, the resource will be recreated.
//
// dropSQL is the full SQL command we will run if we detect that the resource needs updating. It should include
// "IF EXISTS" as it will be attempted even when no previous resource exists. (i.e. new databases)
//
// This function does not check for existence of the resource in the db, thus if you ever delete something manually, it will
// not be recreated until you also delete the corresponding row from schema_hashes.
//
// returns true if schema change was detected
func syncSchema(db *gorm.DB, hashType SchemaHashType, name, desiredSchema, dropSQL string, forceUpdate bool) (bool, error) {

	// Calculate hash of our schema to see if anything has changed.
	hash := sha256.Sum256([]byte(desiredSchema))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])
	vlog := log.WithFields(log.Fields{"name": name, "type": hashType})
	vlog.WithField("hash", hashStr).Debug("generated SHA256 hash")

	currSchemaHash := models.SchemaHash{}
	res := db.Where("type = ? AND name = ?", hashType, name).Find(&currSchemaHash)
	if res.Error != nil {
		vlog.WithError(res.Error).Error("error looking up schema hash")
	}

	var updateRequired bool
	if currSchemaHash.ID == 0 {
		vlog.Debug("no current schema hash in db, creating")
		updateRequired = true
		currSchemaHash = models.SchemaHash{
			Type: string(hashType),
			Name: name,
			Hash: hashStr,
		}
	} else if currSchemaHash.Hash != hashStr {
		vlog.WithField("oldHash", currSchemaHash.Hash).Debug("schema hash has changed, recreating")
		currSchemaHash.Hash = hashStr
		updateRequired = true
	} else if forceUpdate {
		vlog.Debug("schema hash has not changed but a force update was requested, recreating")
		updateRequired = true
	}

	if updateRequired {
		if res := db.Exec(dropSQL); res.Error != nil {
			vlog.WithError(res.Error).Error("error dropping")
			return updateRequired, res.Error
		}

		vlog.Info("creating with latest schema")

		if res := db.Exec(desiredSchema); res.Error != nil {
			log.WithError(res.Error).Error("error creating")
			return updateRequired, res.Error
		}

		if currSchemaHash.ID == 0 {
			if res := db.Create(&currSchemaHash); res.Error != nil {
				vlog.WithError(res.Error).Error("error creating schema hash")
				return updateRequired, res.Error
			}
		} else {
			if res := db.Save(&currSchemaHash); res.Error != nil {
				vlog.WithError(res.Error).Error("error updating schema hash")
				return updateRequired, res.Error
			}
		}
		vlog.Info("schema hash updated")
	} else {
		vlog.Debug("no schema update required")
	}
	return updateRequired, nil
}

func ParseGormLogLevel(logLevel string) (gormlogger.LogLevel, error) {
	switch logLevel {
	case "info":
		return gormlogger.Info, nil
	case "warn":
		return gormlogger.Warn, nil
	case "error":
		return gormlogger.Error, nil
	case "silent":
		return gormlogger.Silent, nil
	default:
		return gormlogger.Info, fmt.Errorf("Unknown gorm LogLevel: %s", logLevel)
	}
}
