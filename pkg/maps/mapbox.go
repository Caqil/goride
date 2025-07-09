package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MapboxProvider struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

func NewMapboxProvider(accessToken string) *MapboxProvider {
	return &MapboxProvider{
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		baseURL:     "https://api.mapbox.com",
	}
}

func (m *MapboxProvider) Geocode(ctx context.Context, address string) (*GeocodeResponse, error) {
	encodedAddress := url.QueryEscape(address)
	apiURL := fmt.Sprintf("%s/geocoding/v5/mapbox.places/%s.json?access_token=%s",
		m.baseURL, encodedAddress, m.accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mapbox API error: %s", string(body))
	}

	var mapboxResp struct {
		Features []struct {
			ID        string    `json:"id"`
			PlaceName string    `json:"place_name"`
			PlaceType []string  `json:"place_type"`
			Center    []float64 `json:"center"`
		} `json:"features"`
	}

	err = json.Unmarshal(body, &mapboxResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	results := make([]GeocodeResult, len(mapboxResp.Features))
	for i, feature := range mapboxResp.Features {
		results[i] = GeocodeResult{
			PlaceID: feature.ID,
			Address: feature.PlaceName,
			Coordinates: Location{
				Latitude:  feature.Center[1],
				Longitude: feature.Center[0],
			},
			Types: feature.PlaceType,
		}
	}

	return &GeocodeResponse{Results: results}, nil
}

func (m *MapboxProvider) ReverseGeocode(ctx context.Context, lat, lng float64) (*GeocodeResponse, error) {
	apiURL := fmt.Sprintf("%s/geocoding/v5/mapbox.places/%f,%f.json?access_token=%s",
		m.baseURL, lng, lat, m.accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mapbox API error: %s", string(body))
	}

	var mapboxResp struct {
		Features []struct {
			ID        string    `json:"id"`
			PlaceName string    `json:"place_name"`
			PlaceType []string  `json:"place_type"`
			Center    []float64 `json:"center"`
		} `json:"features"`
	}

	err = json.Unmarshal(body, &mapboxResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	results := make([]GeocodeResult, len(mapboxResp.Features))
	for i, feature := range mapboxResp.Features {
		results[i] = GeocodeResult{
			PlaceID: feature.ID,
			Address: feature.PlaceName,
			Coordinates: Location{
				Latitude:  feature.Center[1],
				Longitude: feature.Center[0],
			},
			Types: feature.PlaceType,
		}
	}

	return &GeocodeResponse{Results: results}, nil
}

func (m *MapboxProvider) GetDirections(ctx context.Context, request *DirectionsRequest) (*DirectionsResponse, error) {
	coordinates := fmt.Sprintf("%f,%f;%f,%f",
		request.Origin.Longitude, request.Origin.Latitude,
		request.Destination.Longitude, request.Destination.Latitude)

	// Add waypoints
	if len(request.Waypoints) > 0 {
		coords := []string{fmt.Sprintf("%f,%f", request.Origin.Longitude, request.Origin.Latitude)}
		for _, wp := range request.Waypoints {
			coords = append(coords, fmt.Sprintf("%f,%f", wp.Longitude, wp.Latitude))
		}
		coords = append(coords, fmt.Sprintf("%f,%f", request.Destination.Longitude, request.Destination.Latitude))
		coordinates = strings.Join(coords, ";")
	}

	profile := "driving"
	if request.Mode != "" {
		profile = request.Mode
	}

	apiURL := fmt.Sprintf("%s/directions/v5/mapbox/%s/%s?access_token=%s&overview=full&geometries=polyline&steps=true",
		m.baseURL, profile, coordinates, m.accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mapbox API error: %s", string(body))
	}

	var mapboxResp struct {
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Geometry string  `json:"geometry"`
			Legs     []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Steps    []struct {
					Distance    float64   `json:"distance"`
					Duration    float64   `json:"duration"`
					Instruction string    `json:"maneuver.instruction"`
					Maneuver    string    `json:"maneuver.type"`
					Location    []float64 `json:"maneuver.location"`
				} `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}

	err = json.Unmarshal(body, &mapboxResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	routes := make([]Route, len(mapboxResp.Routes))
	for i, route := range mapboxResp.Routes {
		var steps []Step
		for _, leg := range route.Legs {
			for _, step := range leg.Steps {
				steps = append(steps, Step{
					Instructions: step.Instruction,
					Distance: Distance{
						Value: step.Distance,
						Text:  fmt.Sprintf("%.0f m", step.Distance),
					},
					Duration: Duration{
						Value: int(step.Duration),
						Text:  fmt.Sprintf("%.0f s", step.Duration),
					},
					Maneuver: step.Maneuver,
				})
			}
		}

		routes[i] = Route{
			Distance: Distance{
				Value: route.Distance,
				Text:  fmt.Sprintf("%.1f km", route.Distance/1000),
			},
			Duration: Duration{
				Value: int(route.Duration),
				Text:  fmt.Sprintf("%.0f min", route.Duration/60),
			},
			Steps:    steps,
			Polyline: route.Geometry,
		}
	}

	return &DirectionsResponse{Routes: routes}, nil
}

func (m *MapboxProvider) CalculateDistance(ctx context.Context, request *DistanceRequest) (*DistanceResponse, error) {
	// Mapbox doesn't have a direct distance matrix API like Google
	// This would need to be implemented using multiple direction requests
	return nil, fmt.Errorf("CalculateDistance not implemented for Mapbox")
}

func (m *MapboxProvider) GetPlaceDetails(ctx context.Context, placeID string) (*PlaceDetails, error) {
	// Mapbox doesn't have a direct place details API like Google Places
	return nil, fmt.Errorf("GetPlaceDetails not implemented for Mapbox")
}

func (m *MapboxProvider) SearchPlaces(ctx context.Context, request *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	encodedQuery := url.QueryEscape(request.Query)
	apiURL := fmt.Sprintf("%s/geocoding/v5/mapbox.places/%s.json?access_token=%s",
		m.baseURL, encodedQuery, m.accessToken)

	if request.Location.Latitude != 0 && request.Location.Longitude != 0 {
		apiURL += fmt.Sprintf("&proximity=%f,%f", request.Location.Longitude, request.Location.Latitude)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mapbox API error: %s", string(body))
	}

	var mapboxResp struct {
		Features []struct {
			ID        string    `json:"id"`
			PlaceName string    `json:"place_name"`
			PlaceType []string  `json:"place_type"`
			Center    []float64 `json:"center"`
		} `json:"features"`
	}

	err = json.Unmarshal(body, &mapboxResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	results := make([]PlaceResult, len(mapboxResp.Features))
	for i, feature := range mapboxResp.Features {
		results[i] = PlaceResult{
			PlaceID: feature.ID,
			Name:    feature.PlaceName,
			Address: feature.PlaceName,
			Location: Location{
				Latitude:  feature.Center[1],
				Longitude: feature.Center[0],
			},
			Types: feature.PlaceType,
		}
	}

	return &PlaceSearchResponse{Results: results}, nil
}
