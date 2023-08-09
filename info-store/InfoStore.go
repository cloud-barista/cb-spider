// Info <-> MetaDB Store for CB-Spider
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.07.
// by CB-Spider Team, 2019.09.

package infostore

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	icdrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var DB_FILE_PATH string

func init() {
	/*###############################################################*/
	DB_FILE_PATH = os.Getenv("CBSPIDER_ROOT") + "/meta_db/cb-spider.db"
	/*###############################################################*/
}

// KeyValue is a struct for Key-Value pair
// KVList type is used for storing a list of KeyValue with a json format
type KVList []icdrs.KeyValue

func (o *KVList) Scan(src any) error {
	bytes := []byte(src.(string))
	err := json.Unmarshal(bytes, o)
	if err != nil {
		return err
	}
	return nil
}

func (o KVList) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	jsonData, err := json.Marshal(o)
	if err != nil {
		fmt.Println("Failed to serialize to JSON:", err)
		return nil, err
	}
	return string(jsonData), nil
}

// Meta DB Opener
func Open() (*gorm.DB, error) {

	db, err := gorm.Open(sqlite.Open(DB_FILE_PATH), &gorm.Config{})
	if err != nil {
		db = nil
		return nil, err
	}

	return db, nil
}

// Meta DB Closer
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.Close()
	return nil
}

// Insert a Info
func Insert(info interface{}) error {
	db, err := Open()
	if err != nil {
		return err
	}

	defer Close(db)
	if err := db.Save(info).Error; err != nil {
		return err
	}

	return nil
}

// List all Info
func List(infoList interface{}) error {
	db, err := Open()
	if err != nil {
		return err
	}

	defer Close(db)
	if err := db.Find(infoList).Error; err != nil {
		return err
	}

	return nil
}

// Get a Info with name
// input: info(interface of struct), columName(primary key column name), infoName(Info name to Get)
func Get(info interface{}, columName string, infoName string) error {
	db, err := Open()
	if err != nil {
		return err
	}

	defer Close(db)
	if err := db.First(info, columName+" = ?", infoName).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(infoName + ": does not exist!")
		} else {
			return fmt.Errorf(infoName+": %v", err)
		}
	}

	return nil
}

// Delete a Info with name
// input: info(interface of struct), columName(primary key column name), infoName(Info name to Delete)
func Delete(info interface{}, columName string, infoName string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}

	defer Close(db)
	if err := db.First(info, columName+" = ?", infoName).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf(infoName + ": does not exist!")
		} else {
			return false, fmt.Errorf(infoName+": %v", err)
		}
	}

	if err := db.Delete(&info, columName+" = ?", infoName).Error; err != nil {
		return false, fmt.Errorf(infoName+": %v", err)
	}

	return true, nil
}
