package location

import "math"

const RADIUS = float64(6378.16)

//Location is a position of device
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  int32   `json:"altitude"`
}

func Radians(x float64) float64 {
	return x * math.Pi / 180
}

func GetDistance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {

	dlon := Radians(lon2 - lon1)
	dlat := Radians(lat2 - lat1)

	a := (math.Sin(dlat/2) * math.Sin(dlat/2)) + math.Cos(Radians(lat1))*math.Cos(Radians(lat2))*(math.Sin(dlon/2)*math.Sin(dlon/2))
	angle := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return angle * RADIUS
}
