package location

import "math"

// RADIUS is the radius of the Earth in kilometers
const RADIUS = float64(6378.16)

// Location identifies the position of a component by its latitude, longitude and altitude
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  int32   `json:"altitude"`
}

// Radians converts degrees to radians
func Radians(x float64) float64 {
	return x * math.Pi / 180
}

// GetDistance calculates the distance between two points with the Haversine formula
func GetDistance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	dlon := Radians(lon2 - lon1)
	dlat := Radians(lat2 - lat1)
	// Haversine formula
	a := (math.Sin(dlat/2) * math.Sin(dlat/2)) + math.Cos(Radians(lat1))*math.Cos(Radians(lat2))*(math.Sin(dlon/2)*math.Sin(dlon/2))
	angle := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return angle * RADIUS
}
