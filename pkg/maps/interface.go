package maps

import "context"

type MapsProvider interface {
	Geocode(ctx context.Context, address string) (*GeocodeResponse, error)
	ReverseGeocode(ctx context.Context, lat, lng float64) (*GeocodeResponse, error)
	GetDirections(ctx context.Context, request *DirectionsRequest) (*DirectionsResponse, error)
	CalculateDistance(ctx context.Context, request *DistanceRequest) (*DistanceResponse, error)
	GetPlaceDetails(ctx context.Context, placeID string) (*PlaceDetails, error)
	SearchPlaces(ctx context.Context, request *PlaceSearchRequest) (*PlaceSearchResponse, error)
}

type GeocodeResponse struct {
	Results []GeocodeResult `json:"results"`
}

type GeocodeResult struct {
	PlaceID     string   `json:"place_id"`
	Address     string   `json:"formatted_address"`
	Coordinates Location `json:"geometry"`
	Types       []string `json:"types"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DirectionsRequest struct {
	Origin      Location   `json:"origin"`
	Destination Location   `json:"destination"`
	Waypoints   []Location `json:"waypoints,omitempty"`
	Mode        string     `json:"mode"`            // driving, walking, bicycling, transit
	Avoid       []string   `json:"avoid,omitempty"` // tolls, highways, ferries
}

type DirectionsResponse struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Summary  string   `json:"summary"`
	Distance Distance `json:"distance"`
	Duration Duration `json:"duration"`
	Steps    []Step   `json:"steps"`
	Polyline string   `json:"overview_polyline"`
	Bounds   Bounds   `json:"bounds"`
}

type Distance struct {
	Text  string  `json:"text"`
	Value float64 `json:"value"` // in meters
}

type Duration struct {
	Text  string `json:"text"`
	Value int    `json:"value"` // in seconds
}

type Step struct {
	Instructions string   `json:"html_instructions"`
	Distance     Distance `json:"distance"`
	Duration     Duration `json:"duration"`
	StartPoint   Location `json:"start_location"`
	EndPoint     Location `json:"end_location"`
	Polyline     string   `json:"polyline"`
	Maneuver     string   `json:"maneuver"`
}

type Bounds struct {
	Northeast Location `json:"northeast"`
	Southwest Location `json:"southwest"`
}

type DistanceRequest struct {
	Origins      []Location `json:"origins"`
	Destinations []Location `json:"destinations"`
	Mode         string     `json:"mode"`
	Units        string     `json:"units"` // metric, imperial
}

type DistanceResponse struct {
	Rows []DistanceRow `json:"rows"`
}

type DistanceRow struct {
	Elements []DistanceElement `json:"elements"`
}

type DistanceElement struct {
	Distance Distance `json:"distance"`
	Duration Duration `json:"duration"`
	Status   string   `json:"status"`
}

type PlaceSearchRequest struct {
	Query    string   `json:"query"`
	Location Location `json:"location,omitempty"`
	Radius   int      `json:"radius,omitempty"`
	Type     string   `json:"type,omitempty"`
}

type PlaceSearchResponse struct {
	Results []PlaceResult `json:"results"`
}

type PlaceResult struct {
	PlaceID    string   `json:"place_id"`
	Name       string   `json:"name"`
	Address    string   `json:"formatted_address"`
	Location   Location `json:"geometry"`
	Rating     float64  `json:"rating"`
	Types      []string `json:"types"`
	PriceLevel int      `json:"price_level"`
}

type PlaceDetails struct {
	PlaceID          string        `json:"place_id"`
	Name             string        `json:"name"`
	Address          string        `json:"formatted_address"`
	Location         Location      `json:"geometry"`
	PhoneNumber      string        `json:"formatted_phone_number"`
	Website          string        `json:"website"`
	Rating           float64       `json:"rating"`
	UserRatingsTotal int           `json:"user_ratings_total"`
	Types            []string      `json:"types"`
	Photos           []Photo       `json:"photos"`
	OpeningHours     *OpeningHours `json:"opening_hours"`
}

type Photo struct {
	PhotoReference string `json:"photo_reference"`
	Height         int    `json:"height"`
	Width          int    `json:"width"`
}

type OpeningHours struct {
	OpenNow bool     `json:"open_now"`
	Periods []Period `json:"periods"`
}

type Period struct {
	Open  TimeOfWeek `json:"open"`
	Close TimeOfWeek `json:"close"`
}

type TimeOfWeek struct {
	Day  int    `json:"day"`
	Time string `json:"time"`
}
