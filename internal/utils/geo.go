package utils

import (
	"fmt"
	"math"
)

type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Bounds struct {
	Northeast Point `json:"northeast"`
	Southwest Point `json:"southwest"`
}

type Polygon []Point

func (p Point) ToCoordinates() []float64 {
	return []float64{p.Lng, p.Lat}
}

func (p Point) String() string {
	return fmt.Sprintf("%.6f,%.6f", p.Lat, p.Lng)
}

func NewPointFromCoordinates(coordinates []float64) Point {
	if len(coordinates) >= 2 {
		return Point{Lat: coordinates[1], Lng: coordinates[0]}
	}
	return Point{}
}

func IsValidCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func NormalizeCoordinates(lat, lng float64) (float64, float64) {
	// Normalize latitude to [-90, 90]
	if lat > 90 {
		lat = 90
	} else if lat < -90 {
		lat = -90
	}

	// Normalize longitude to [-180, 180]
	for lng > 180 {
		lng -= 360
	}
	for lng < -180 {
		lng += 360
	}

	return lat, lng
}

func CalculateCenter(points []Point) Point {
	if len(points) == 0 {
		return Point{}
	}

	var totalLat, totalLng float64
	for _, point := range points {
		totalLat += point.Lat
		totalLng += point.Lng
	}

	return Point{
		Lat: totalLat / float64(len(points)),
		Lng: totalLng / float64(len(points)),
	}
}

func CalculateBounds(points []Point) *Bounds {
	if len(points) == 0 {
		return nil
	}

	minLat, maxLat := points[0].Lat, points[0].Lat
	minLng, maxLng := points[0].Lng, points[0].Lng

	for _, point := range points {
		if point.Lat < minLat {
			minLat = point.Lat
		}
		if point.Lat > maxLat {
			maxLat = point.Lat
		}
		if point.Lng < minLng {
			minLng = point.Lng
		}
		if point.Lng > maxLng {
			maxLng = point.Lng
		}
	}

	return &Bounds{
		Northeast: Point{Lat: maxLat, Lng: maxLng},
		Southwest: Point{Lat: minLat, Lng: minLng},
	}
}

func IsPointInPolygon(point Point, polygon Polygon) bool {
	if len(polygon) < 3 {
		return false
	}

	x, y := point.Lng, point.Lat
	inside := false
	var xinters float64 // Declare xinters here with function scope

	p1x, p1y := polygon[0].Lng, polygon[0].Lat
	for i := 1; i <= len(polygon); i++ {
		p2x, p2y := polygon[i%len(polygon)].Lng, polygon[i%len(polygon)].Lat

		if y > math.Min(p1y, p2y) {
			if y <= math.Max(p1y, p2y) {
				if x <= math.Max(p1x, p2x) {
					if p1y != p2y {
						xinters = (y-p1y)/(p2y-p1y)*(p2x-p1x) + p1x
					} else {
						xinters = p1x // Assign a default value if p1y == p2y
					}
					if p1x == p2x || x <= xinters {
						inside = !inside
					}
				}
			}
		}
		p1x, p1y = p2x, p2y
	}

	return inside
}

func IsPointInCircle(point Point, center Point, radiusKM float64) bool {
	distance := CalculateDistance(center.Lat, center.Lng, point.Lat, point.Lng)
	return distance <= radiusKM
}

func EncodePolyline(points []Point) string {
	// Simplified polyline encoding - in production, use a proper library
	encoded := ""
	prevLat, prevLng := 0, 0

	for _, point := range points {
		lat := int(point.Lat * 1e5)
		lng := int(point.Lng * 1e5)

		dLat := lat - prevLat
		dLng := lng - prevLng

		encoded += encodeSignedNumber(dLat)
		encoded += encodeSignedNumber(dLng)

		prevLat = lat
		prevLng = lng
	}

	return encoded
}

func encodeSignedNumber(num int) string {
	sgn_num := num << 1
	if num < 0 {
		sgn_num = ^sgn_num
	}
	return encodeNumber(sgn_num)
}

func encodeNumber(num int) string {
	encoded := ""
	for num >= 0x20 {
		encoded += string(rune((0x20 | (num & 0x1f)) + 63))
		num >>= 5
	}
	encoded += string(rune(num + 63))
	return encoded
}
