// Meta DB Opener for Cloud Driver Info Manager.
// Cloud Driver Info Manager is a module of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.07.

package metadb

import (
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open() (*gorm.DB, error) {
	/*############################################################*/
	dbPath := os.Getenv("CBSPIDER_ROOT") + "/meta_db/cb-spider.db"
	/*############################################################*/

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		db = nil
		return nil, err
	}

	return db, nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.Close()
	return nil
}
