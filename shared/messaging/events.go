package messaging

import (
	pbd "ride-sharing/shared/proto/driver"
	pbt "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue  = "find_available_drivers"
	DriverCmdTripRequestQueue  = "driver_cmd_trip_request"
	DriverCmdTripResponseQueue = "driver_trip_response"
	NotifyNoDriversFoundQueue  = "notify_no_drivers_found"
	NotifyDriverAssignQueue    = "notify_driver_assign_queue"
)

type TripEventData struct {
	Trip *pbt.Trip `json:"trip"`
}

type DriverTripResponseData struct {
	Driver  *pbd.Driver `json:"driver"`
	TripID  string      `json:"tripID"`
	RiderID string      `json:"riderID"`
}
