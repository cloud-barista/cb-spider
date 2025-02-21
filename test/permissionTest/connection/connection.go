package connection

import "os"

type Connection struct {
	Name     string
	CanWrite bool
	CanRead  bool
}

var Connections = map[string]Connection{
	"rw-conn":             {Name: os.Getenv("RW_CONN_NAME"), CanWrite: true, CanRead: true},
    "rw-conn2":            {Name: os.Getenv("RW_CONN2_NAME"), CanWrite: true, CanRead: true},
    "readonly-conn":       {Name: os.Getenv("READONLY_CONN_NAME"), CanWrite: false, CanRead: true},
    "non-permission-conn": {Name: os.Getenv("NON_PERMISSION_CONN_NAME"), CanWrite: false, CanRead: false},
}