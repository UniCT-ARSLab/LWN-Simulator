package geometry

import "math"

const earthRadiusMeters = 6_371_000.0

func DistanceMeters(latitudeA, longitudeA, latitudeB, longitudeB float64) float64 {
	latitudeDelta := degreesToRadians(latitudeB - latitudeA)
	longitudeDelta := degreesToRadians(longitudeB - longitudeA)
	latitudeARadians := degreesToRadians(latitudeA)
	latitudeBRadians := degreesToRadians(latitudeB)

	haversine := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(latitudeARadians)*math.Cos(latitudeBRadians)*
			math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	haversine = math.Min(1, math.Max(0, haversine))

	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(haversine))
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
