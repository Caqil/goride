package maps

import (
	"context"
	"fmt"

	"googlemaps.github.io/maps"
)

type GoogleMapsProvider struct {
	client *maps.Client
}

func NewGoogleMapsProvider(apiKey string) (*GoogleMapsProvider, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Maps client: %w", err)
	}

	return &GoogleMapsProvider{
		client: client,
	}, nil
}

func (g *GoogleMapsProvider) Geocode(ctx context.Context, address string) (*GeocodeResponse, error) {
	req := &maps.GeocodingRequest{
		Address: address,
	}

	resp, err := g.client.Geocode(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("geocoding failed: %w", err)
	}

	results := make([]GeocodeResult, len(resp))
	for i, result := range resp {
		results[i] = GeocodeResult{
			PlaceID: result.PlaceID,
			Address: result.FormattedAddress,
			Coordinates: Location{
				Latitude:  result.Geometry.Location.Lat,
				Longitude: result.Geometry.Location.Lng,
			},
			Types: result.Types,
		}
	}

	return &GeocodeResponse{Results: results}, nil
}

func (g *GoogleMapsProvider) ReverseGeocode(ctx context.Context, lat, lng float64) (*GeocodeResponse, error) {
	req := &maps.GeocodingRequest{
		LatLng: &maps.LatLng{Lat: lat, Lng: lng},
	}

	resp, err := g.client.ReverseGeocode(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("reverse geocoding failed: %w", err)
	}

	results := make([]GeocodeResult, len(resp))
	for i, result := range resp {
		results[i] = GeocodeResult{
			PlaceID: result.PlaceID,
			Address: result.FormattedAddress,
			Coordinates: Location{
				Latitude:  result.Geometry.Location.Lat,
				Longitude: result.Geometry.Location.Lng,
			},
			Types: result.Types,
		}
	}

	return &GeocodeResponse{Results: results}, nil
}

func (g *GoogleMapsProvider) GetDirections(ctx context.Context, request *DirectionsRequest) (*DirectionsResponse, error) {
	req := &maps.DirectionsRequest{
		Origin:      fmt.Sprintf("%f,%f", request.Origin.Latitude, request.Origin.Longitude),
		Destination: fmt.Sprintf("%f,%f", request.Destination.Latitude, request.Destination.Longitude),
		Mode:        maps.Mode(request.Mode),
	}

	// Add waypoints if any
	if len(request.Waypoints) > 0 {
		waypoints := make([]string, len(request.Waypoints))
		for i, wp := range request.Waypoints {
			waypoints[i] = fmt.Sprintf("%f,%f", wp.Latitude, wp.Longitude)
		}
		req.Waypoints = waypoints
	}

	// Add avoid preferences
	if len(request.Avoid) > 0 {
		avoid := make([]maps.Avoid, len(request.Avoid))
		for i, a := range request.Avoid {
			avoid[i] = maps.Avoid(a)
		}
		req.Avoid = avoid
	}

	resp, _, err := g.client.Directions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("directions request failed: %w", err)
	}

	routes := make([]Route, len(resp))
	for i, route := range resp {
		steps := make([]Step, len(route.Legs[0].Steps))
		for j, step := range route.Legs[0].Steps {
			steps[j] = Step{
				Instructions: step.HTMLInstructions,
				Distance: Distance{
					Text:  step.Distance.HumanReadable,
					Value: float64(step.Distance.Meters),
				},
				Duration: Duration{
					Text:  step.Duration.String(),
					Value: int(step.Duration.Seconds()),
				},
				StartPoint: Location{
					Latitude:  step.StartLocation.Lat,
					Longitude: step.StartLocation.Lng,
				},
				EndPoint: Location{
					Latitude:  step.EndLocation.Lat,
					Longitude: step.EndLocation.Lng,
				},
				Polyline: step.Polyline.Points,
			}
		}

		routes[i] = Route{
			Summary: route.Summary,
			Distance: Distance{
				Text:  route.Legs[0].Distance.HumanReadable,
				Value: float64(route.Legs[0].Distance.Meters),
			},
			Duration: Duration{
				Text:  route.Legs[0].Duration.String(),
				Value: int(route.Legs[0].Duration.Seconds()),
			},
			Steps:    steps,
			Polyline: route.OverviewPolyline.Points,
			Bounds: Bounds{
				Northeast: Location{
					Latitude:  route.Bounds.NorthEast.Lat,
					Longitude: route.Bounds.NorthEast.Lng,
				},
				Southwest: Location{
					Latitude:  route.Bounds.SouthWest.Lat,
					Longitude: route.Bounds.SouthWest.Lng,
				},
			},
		}
	}

	return &DirectionsResponse{Routes: routes}, nil
}

func (g *GoogleMapsProvider) CalculateDistance(ctx context.Context, request *DistanceRequest) (*DistanceResponse, error) {
	origins := make([]string, len(request.Origins))
	for i, origin := range request.Origins {
		origins[i] = fmt.Sprintf("%f,%f", origin.Latitude, origin.Longitude)
	}

	destinations := make([]string, len(request.Destinations))
	for i, dest := range request.Destinations {
		destinations[i] = fmt.Sprintf("%f,%f", dest.Latitude, dest.Longitude)
	}

	req := &maps.DistanceMatrixRequest{
		Origins:      origins,
		Destinations: destinations,
		Mode:         maps.Mode(request.Mode),
		Units:        maps.Units(request.Units),
	}

	resp, err := g.client.DistanceMatrix(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("distance matrix request failed: %w", err)
	}

	rows := make([]DistanceRow, len(resp.Rows))
	for i, row := range resp.Rows {
		elements := make([]DistanceElement, len(row.Elements))
		for j, element := range row.Elements {
			elements[j] = DistanceElement{
				Distance: Distance{
					Text:  element.Distance.HumanReadable,
					Value: float64(element.Distance.Meters),
				},
				Duration: Duration{
					Text:  element.Duration.String(),
					Value: int(element.Duration.Seconds()),
				},
				Status: string(element.Status),
			}
		}
		rows[i] = DistanceRow{Elements: elements}
	}

	return &DistanceResponse{Rows: rows}, nil
}

func (g *GoogleMapsProvider) GetPlaceDetails(ctx context.Context, placeID string) (*PlaceDetails, error) {
	req := &maps.PlaceDetailsRequest{
		PlaceID: placeID,
		Fields: []maps.PlaceDetailsFieldMask{
			maps.PlaceDetailsFieldMaskPlaceID,
			maps.PlaceDetailsFieldMaskName,
			maps.PlaceDetailsFieldMaskFormattedAddress,
			maps.PlaceDetailsFieldMaskGeometry,
			maps.PlaceDetailsFieldMaskFormattedPhoneNumber,
			maps.PlaceDetailsFieldMaskWebsite,
			// maps.PlaceDetailsFieldMaskRating, // Removed because it does not exist in the maps package
			maps.PlaceDetailsFieldMaskUserRatingsTotal,
			maps.PlaceDetailsFieldMaskTypes,
			maps.PlaceDetailsFieldMaskPhotos,
			maps.PlaceDetailsFieldMaskOpeningHours,
		},
	}

	resp, err := g.client.PlaceDetails(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("place details request failed: %w", err)
	}

	details := &PlaceDetails{
		PlaceID: resp.PlaceID,
		Name:    resp.Name,
		Address: resp.FormattedAddress,
		Location: Location{
			Latitude:  resp.Geometry.Location.Lat,
			Longitude: resp.Geometry.Location.Lng,
		},
		PhoneNumber:      resp.FormattedPhoneNumber,
		Website:          resp.Website,
		Rating:           float64(resp.Rating),
		UserRatingsTotal: resp.UserRatingsTotal,
		Types:            resp.Types,
	}

	// Convert photos
	if len(resp.Photos) > 0 {
		details.Photos = make([]Photo, len(resp.Photos))
		for i, photo := range resp.Photos {
			details.Photos[i] = Photo{
				PhotoReference: photo.PhotoReference,
				Height:         photo.Height,
				Width:          photo.Width,
			}
		}
	}

	// Convert opening hours
	if resp.OpeningHours != nil {
		openNow := false
		if resp.OpeningHours.OpenNow != nil {
			openNow = *resp.OpeningHours.OpenNow
		}
		details.OpeningHours = &OpeningHours{
			OpenNow: openNow,
		}

		if len(resp.OpeningHours.Periods) > 0 {
			details.OpeningHours.Periods = make([]Period, len(resp.OpeningHours.Periods))
			for i, period := range resp.OpeningHours.Periods {
				details.OpeningHours.Periods[i] = Period{
					Open: TimeOfWeek{
						Day:  int(period.Open.Day),
						Time: period.Open.Time,
					},
					Close: TimeOfWeek{
						Day:  int(period.Close.Day),
						Time: period.Close.Time,
					},
				}
			}
		}
	}

	return details, nil
}

func (g *GoogleMapsProvider) SearchPlaces(ctx context.Context, request *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	req := &maps.TextSearchRequest{
		Query: request.Query,
	}

	if request.Location.Latitude != 0 && request.Location.Longitude != 0 {
		req.Location = &maps.LatLng{
			Lat: request.Location.Latitude,
			Lng: request.Location.Longitude,
		}
	}

	if request.Radius > 0 {
		req.Radius = uint(request.Radius)
	}

	if request.Type != "" {
		req.Type = maps.PlaceType(request.Type)
	}

	resp, err := g.client.TextSearch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("place search request failed: %w", err)
	}

	results := make([]PlaceResult, len(resp.Results))
	for i, result := range resp.Results {
		results[i] = PlaceResult{
			PlaceID: result.PlaceID,
			Name:    result.Name,
			Address: result.FormattedAddress,
			Location: Location{
				Latitude:  result.Geometry.Location.Lat,
				Longitude: result.Geometry.Location.Lng,
			},
			Rating:     float64(result.Rating),
			Types:      result.Types,
			PriceLevel: result.PriceLevel,
		}
	}

	return &PlaceSearchResponse{Results: results}, nil
}
