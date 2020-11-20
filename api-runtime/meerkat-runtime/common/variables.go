// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package common


const (

	/////////////////////////// StatusZone Table
	MaxSpiders = "20"
	TransCount = "3"
	TimerTime = 5 // seconds (HeartBeat)
	ChildKatCallTimeout = 3000 // milliseconds

	//// metadata base position of Table
	// Status Table
	StatusSheetName = "StatusZone"
	// X
	StatusRowLockX = "b"
	StatusSpiderIDX = "c"
	StatusStatusX = "d"
	StatusTimeX = "e"
	StatusCountX = "f"
	// Y
	StatusTableY = "5"

	// Response Table: TBD

	/////////////////////////// CommandZone Table
        //// metadata base position of Table
        // Command Queue Table
	MaxCommands = "10"

	// Command Table
        CommandSheetName = "CommandZone"
        // X
        CommandIDX = "b"
        CommandTypeX = "c"
        CommandCMDX = "d"

        CommandSpiderIDX = "g"
        CommandResultNow = "h"
        CommandResultBefore = "i"
        CommandResultBeforeBefore = "j"
	CommandResultTimeX = "k"
        // Y
        CommandTableY = "5"

)

