package utils

import (
	"math"
)

func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	return haversineDistance(lat1, lon1, lat2, lon2)
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Differences
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance in kilometers
	distance := EarthRadiusKM * c
	return distance
}

func CalculateDistanceInMiles(lat1, lon1, lat2, lon2 float64) float64 {
	distanceKM := CalculateDistance(lat1, lon1, lat2, lon2)
	return distanceKM * 0.621371 // Convert to miles
}

func IsWithinRadius(centerLat, centerLon, pointLat, pointLon, radiusKM float64) bool {
	distance := CalculateDistance(centerLat, centerLon, pointLat, pointLon)
	return distance <= radiusKM
}

func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	bearing = math.Mod(bearing+360, 360)

	return bearing
}

func EstimateETAMinutes(distanceKM float64, averageSpeedKMH float64) int {
	if averageSpeedKMH <= 0 {
		averageSpeedKMH = 30 // Default city speed
	}
	
	timeHours := distanceKM / averageSpeedKMH
	timeMinutes := timeHours * 60
	
	return int(math.Ceil(timeMinutes))
}