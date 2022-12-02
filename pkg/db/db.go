package db

import (
	"log"
	"os"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

////////////
// Models //
////////////

// Monitor refers to monitors that are being run by the server.
type Monitor struct {
	gorm.Model

	Name        string
	LastChecked time.Time
	LastStatus  string
}

// User refers to any user of the application that must be tracked.
type User struct {
	gorm.Model

	Token string // how they link their devices to our platform.
}

// Product refer to our hardware units.
type Product struct {
	gorm.Model

	Code      string
	Price     uint
	Inventory uint
	Available bool
}

// Event refers to Alert, Notification, and other Events in our
// system that we want to keep a log of in a more structured way.
type Event struct {
	gorm.Model

	Code    uint   // error code, status code, etc...
	Kind    string // alert, notification, warning, etc...
	Message string // the message the Event contained, e.g. the alert's value
	Source  string // foreign key to a Monitor.
	Payload datatypes.JSON
}

////////////////
// CONNECTION //
////////////////

// New returns a new gorm.DB instance.
// We are going to rely on Gorm for this project, we're considering
// this a dependency now.
func New() *gorm.DB {
	dsn := os.Getenv("POSTGRES_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to get pg connection: %v", err)
	}
	db.AutoMigrate(&Monitor{}, &User{}, &Product{}, &Event{})
	return db
}
