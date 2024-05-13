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
	"gorm.io/gorm/logger"

	cblogger "github.com/cloud-barista/cb-log"
	icdrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblog *logrus.Logger

var DB_FILE_PATH string

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")

	/*###############################################################*/
	DB_PATH := os.Getenv("CBSPIDER_ROOT") + "/meta_db"
	DB_FILE_PATH = DB_PATH + "/cb-spider.db"
	/*###############################################################*/

	// if no path, makes it
	_, err := os.Stat(DB_PATH)
	if os.IsNotExist(err) {
		err := os.Mkdir(DB_PATH, 0755)
		if err != nil {
			cblog.Fatal(err)
			return
		}
	}
}

func Ping() error {
	// check database connection
	db, err := Open()
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %s", err.Error())
	}

	defer Close(db)
	err = sqlDB.Ping()
	if err != nil {
		return fmt.Errorf("Database not responding: %s", err.Error())
	}

	return nil
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
		return nil, err
	}
	return string(jsonData), nil
}

type AZList []string

func (o *AZList) Scan(src any) error {
	bytes := []byte(src.(string))
	err := json.Unmarshal(bytes, o)
	if err != nil {
		return err
	}
	return nil
}

func (o AZList) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	jsonData, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return string(jsonData), nil
}

// Meta DB Opener
func Open() (*gorm.DB, error) {

	// Turn-on error logs of gorm: db, err := gorm.Open(sqlite.Open(DB_FILE_PATH), &gorm.Config{})
	db, err := gorm.Open(sqlite.Open(DB_FILE_PATH), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent)})
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

//////////////////////////////////
// API for Tables with single key
// DriverInfo, CredentialInfo, ...
//////////////////////////////////

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

// Get a Info with a condition
func Get(info interface{}, columnName string, columnValue string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	if err := db.First(&info, columnName+" = ?", columnValue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(columnValue + ": does not exist!")
		} else {
			return fmt.Errorf(columnValue+": %v", err)
		}
	}

	return nil
}

// Check if a Info exists with a condition
func Has(info interface{}, columnName string, columnValue string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}

	defer Close(db)
	if err := db.First(&info, columnName+" = ?", columnValue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		} else {
			return false, fmt.Errorf(columnValue+": %v", err)
		}
	}
	return true, nil
}

// Delete a Info with a condition
func Delete(info interface{}, columName string, columnValue string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}

	defer Close(db)

	if err := db.Delete(&info, columName+" = ?", columnValue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf(columnValue + ": does not exist!")
		} else {
			return false, fmt.Errorf(columnValue+": %v", err)
		}
	}

	return true, nil
}

// ////////////////////////////////
// API for Tables with composite key
// VPCInfo, SecurityInfo, ...
// composit key ex) Connection Name + Resource Name
// ////////////////////////////////

// List all Info with a condition(ex. Conneciton Name)
func ListByCondition(infoList interface{}, columnName string, columnValue string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	if err := db.Find(infoList, columnName+" = ?", columnValue).Error; err != nil {
		return err
	}

	return nil
}

// List all Info with two conditions(ex. Conneciton Name and Owner VPC Name)
// Used for SubnetInfoList, ...
func ListByConditions(infoList interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	if err := db.Find(infoList, columnName1+" = ? AND "+columnName2+" = ?", columnValue1, columnValue2).Error; err != nil {
		return err
	}

	return nil
}

// Get a Info with two conditions(Conneciton Name, Resource NameId)
func GetByConditions(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	if err := db.Where(columnName1+" = ? AND "+columnName2+" = ?", columnValue1, columnValue2).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(columnValue1 + ", " + columnValue2 + ": does not exist!")
		} else {
			return fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return nil
}

// GetByConditionAndContain finds an entry that matches the given conditions:
// - columnName1 must equal columnValue1
// - columnName2 must contain columnContainValue2
func GetByConditionAndContain(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnContainValue2 string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	// Check if columnValue3 is empty and handle accordingly
	if columnContainValue2 == "" {
		return fmt.Errorf("%s, %s: does not exist!", columnValue1, columnContainValue2)
	}

	// Use LIKE operator for columnName2 to check if it contains columnContainValue2
	query := fmt.Sprintf("%s = ? AND %s LIKE ?", columnName1, columnName2)
	if err := db.Where(query, columnValue1, "%"+columnContainValue2+"%").First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("%s, %s: does not exist!", columnValue1, columnContainValue2)
		} else {
			return fmt.Errorf("%s, %s: %v", columnValue1, columnContainValue2, err)
		}
	}

	return nil
}

// Get a Info with three conditions(Conneciton Name, Resource NameId, Owner VPC Name)
func GetBy3Conditions(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string,
	columnName3 string, columnValue3 string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	if err := db.Where(columnName1+" = ? AND "+columnName2+" = ? AND "+columnName3+" = ?", columnValue1, columnValue2, columnValue3).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(columnValue1 + ", " + columnValue2 + ": does not exist!")
		} else {
			return fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return nil
}

// Get a Info with a condition(Conneciton Name) and contains(contained_text)
func GetByContain(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	if err := db.Where(columnName1+" = ? AND "+columnName2+" LIKE ?",
		columnValue1, fmt.Sprintf("%%%s%%", columnValue2)).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(columnValue1 + ", " + columnValue2 + ": does not exist!")
		} else {
			return fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return nil
}

// Get a Info with two conditions(Conneciton Name, Resource NameId) and contain(contained_text)
func GetByConditionsAndContain(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string,
	columnName3 string, columnValue3 string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer Close(db)

	// Check if columnValue3 is empty and handle accordingly
	if columnValue3 == "" {
		return fmt.Errorf(columnValue1 + ", " + columnValue2 + ", " + columnValue3 + ": does not exist!")
	}

	if err := db.Where(columnName1+" = ? AND "+columnName2+" = ? AND "+columnName3+" LIKE ?",
		columnValue1, columnValue2, fmt.Sprintf("%%%s%%", columnValue3)).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(columnValue1 + ", " + columnValue2 + ", " + columnValue3 + ": does not exist!")
		} else {
			return fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return nil
}

// Check if a Info exists with two conditions(Conneciton Name, Resource NameId)
func HasByConditions(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}
	defer Close(db)

	if err := db.Where(columnName1+" = ? AND "+columnName2+" = ?", columnValue1, columnValue2).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		} else {
			return false, fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return true, nil
}

// Check if a Info exists with three conditions(Conneciton Name, Resource NameId, Owner vpc name)
func HasBy3Conditions(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string,
	columnName3 string, columnValue3 string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}
	defer Close(db)

	if err := db.Where(columnName1+" = ? AND "+columnName2+" = ? AND "+columnName3+" = ?", columnValue1, columnValue2, columnValue3).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		} else {
			return false, fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return true, nil
}

// Delete all Infos with two conditions
// ex) Conneciton Name, Resource Name
// ex) Conneciton Name, Owner VPC Name
func DeleteByConditions(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}

	defer Close(db)

	if err := db.Delete(&info, columnName1+" = ? AND "+columnName2+" = ?", columnValue1, columnValue2).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf(columnValue1 + ", " + columnValue2 + ": does not exist!")
		} else {
			return false, fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return true, nil
}

// Delete all Infos with three conditions
// ex) Conneciton Name, Resource Name, Owner VPC Name
func DeleteBy3Conditions(info interface{}, columnName1 string, columnValue1 string, columnName2 string, columnValue2 string,
	columnName3 string, columnValue3 string) (bool, error) {
	db, err := Open()
	if err != nil {
		return false, err
	}

	defer Close(db)

	if err := db.Delete(&info, columnName1+" = ? AND "+columnName2+" = ?  AND "+columnName3+" = ?", columnValue1, columnValue2, columnValue3).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf(columnValue1 + ", " + columnValue2 + ": does not exist!")
		} else {
			return false, fmt.Errorf(columnValue1+", "+columnValue2+": %v", err)
		}
	}

	return true, nil
}

// CountAllConnections counts all entries in the connection_config_infos table
// info is a pointer to a ConnectionConfigInfo struct
func CountAllConnections(info interface{}) (int64, error) {
	db, err := Open()
	if err != nil {
		return 0, err
	}
	defer Close(db)

	var count int64
	if err := db.Model(&info).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// CountConnectionsByProvider counts entries in the connection_config_infos table by provider_name
// info is a pointer to a ConnectionConfigInfo struct
func CountConnectionsByProvider(info interface{}, providerName string) (int64, error) {
	db, err := Open()
	if err != nil {
		return 0, err
	}
	defer Close(db)

	var count int64
	if err := db.Model(&info).Where("provider_name = ?", providerName).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// CountAllNameIDs counts all name_ids using the model type passed as info
func CountAllNameIDs(info interface{}) (int64, error) {
	db, err := Open()
	if err != nil {
		return 0, err
	}
	defer Close(db)

	var count int64
	if err := db.Model(&info).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// CountNameIDsByConnection counts name_ids in a model filtered by connection_name
func CountNameIDsByConnection(info interface{}, connectionName string) (int64, error) {
	db, err := Open()
	if err != nil {
		return 0, err
	}
	defer Close(db)

	var count int64
	if err := db.Model(&info).Where("connection_name = ?", connectionName).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
