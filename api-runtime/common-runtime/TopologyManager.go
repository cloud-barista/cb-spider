// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Topology Layout Manager — multi-version layout persistence in Spider MetaDB.
//
// by CB-Spider Team, 2025.06.

package commonruntime

import (
	"fmt"
	"strings"
	"time"

	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// TopologyLayoutInfo stores one named version of a topology layout.
// Primary key is "connectionName::versionName" to avoid composite-PK issues with SQLite/GORM.
type TopologyLayoutInfo struct {
	ID              string `gorm:"primaryKey"`   // "connectionName::versionName"
	ConnectionName  string `gorm:"index;not null"`
	VersionName     string `gorm:"not null"`
	LayoutJSON      string `gorm:"type:text"`
	ThumbnailBase64 string `gorm:"type:text"`
	SavedAt         string `gorm:"not null;default:''"`
}

func (TopologyLayoutInfo) TableName() string { return "topology_layout_infos" }

func makeTopoID(connectionName, versionName string) string {
	return connectionName + "::" + versionName
}

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	defer infostore.Close(db)

	// Drop old table if schema is incompatible (old single-PK or composite-PK schema).
	if db.Migrator().HasTable(&TopologyLayoutInfo{}) {
		if !db.Migrator().HasColumn(&TopologyLayoutInfo{}, "id") {
			cblog.Info("TopologyManager: recreating topology_layout_infos with new schema")
			db.Migrator().DropTable(&TopologyLayoutInfo{})
		}
	}
	db.AutoMigrate(&TopologyLayoutInfo{})
}

// SaveTopologyLayout upserts a named layout version for a connection.
func SaveTopologyLayout(connectionName, versionName, layoutJSON, thumbnailBase64 string) error {
	connectionName = strings.TrimSpace(connectionName)
	versionName    = strings.TrimSpace(versionName)
	if connectionName == "" { return fmt.Errorf("connectionName is empty") }
	if versionName    == "" { return fmt.Errorf("versionName is empty") }
	if layoutJSON     == "" { return fmt.Errorf("layoutJSON is empty") }

	id := makeTopoID(connectionName, versionName)
	now := time.Now().UTC().Format(time.RFC3339)

	db, err := infostore.Open()
	if err != nil { return err }
	defer infostore.Close(db)

	var existing TopologyLayoutInfo
	if db.Where("id = ?", id).First(&existing).Error == nil {
		// Update existing record
		return db.Model(&existing).Updates(map[string]interface{}{
			"layout_json":      layoutJSON,
			"thumbnail_base64": thumbnailBase64,
			"saved_at":         now,
		}).Error
	}
	// Insert new record
	info := TopologyLayoutInfo{
		ID:              id,
		ConnectionName:  connectionName,
		VersionName:     versionName,
		LayoutJSON:      layoutJSON,
		ThumbnailBase64: thumbnailBase64,
		SavedAt:         now,
	}
	return db.Create(&info).Error
}

// ListTopologyLayouts returns all saved versions for a connection (ordered newest first).
func ListTopologyLayouts(connectionName string) ([]TopologyLayoutInfo, error) {
	connectionName = strings.TrimSpace(connectionName)
	if connectionName == "" { return nil, fmt.Errorf("connectionName is empty") }

	db, err := infostore.Open()
	if err != nil { return nil, err }
	defer infostore.Close(db)

	var list []TopologyLayoutInfo
	result := db.Where("connection_name = ?", connectionName).
		Order("saved_at desc").Find(&list)
	return list, result.Error
}

// GetTopologyLayout returns one named version (full LayoutJSON included).
func GetTopologyLayout(connectionName, versionName string) (TopologyLayoutInfo, error) {
	id := makeTopoID(strings.TrimSpace(connectionName), strings.TrimSpace(versionName))

	db, err := infostore.Open()
	if err != nil { return TopologyLayoutInfo{}, err }
	defer infostore.Close(db)

	var info TopologyLayoutInfo
	if db.Where("id = ?", id).First(&info).Error != nil {
		return TopologyLayoutInfo{}, fmt.Errorf("version '%s' not found", versionName)
	}
	return info, nil
}

// DeleteTopologyLayout deletes one named version.
func DeleteTopologyLayout(connectionName, versionName string) error {
	id := makeTopoID(strings.TrimSpace(connectionName), strings.TrimSpace(versionName))

	db, err := infostore.Open()
	if err != nil { return err }
	defer infostore.Close(db)

	return db.Where("id = ?", id).Delete(&TopologyLayoutInfo{}).Error
}
