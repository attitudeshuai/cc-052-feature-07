package router

import (
	"cc-052/internal/handler"
	"testing"

	"github.com/go-redis/redis/v8"
)

// The router must build without gin route-conflict panics.
func TestSetupBuilds(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer rdb.Close()
	r := Setup(
		&handler.FarmHandler{}, &handler.PlotHandler{}, &handler.BatchHandler{},
		&handler.ActivityHandler{}, &handler.InspectionHandler{}, &handler.TraceCodeHandler{},
		&handler.HealthHandler{}, rdb,
	)
	found := false
	for _, ri := range r.Routes() {
		if ri.Method == "GET" && ri.Path == "/api/v1/batches/:id/codes/stats" {
			found = true
		}
	}
	if !found {
		t.Fatal("stats route not registered")
	}
}
