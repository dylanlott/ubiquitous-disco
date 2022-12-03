// @Title
// @Description
// @Author
// @Update

package server

import (
	"fmt"
	"net/http"

	"github.com/fly-apps/go-example/pkg/alerts"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"gorm.io/gorm"
)

// S holds all of the relevant pieces together
// for our monitoring service.
type S struct {
	db     *gorm.DB
	influx influxdb2.Client
	siren  *alerts.Siren
	srv    *http.Server
}

func New() (*S, error) {
	return nil, fmt.Errorf("not impl")
}
