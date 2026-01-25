package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type service struct {
	repo domain.TripRepository
}

func NewService(repo domain.TripRepository) *service {
	return &service{repo: repo}
}

func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	trip := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   &trip.TripDriver{},
	}

	return s.repo.CreateTrip(ctx, trip)
}

func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMApiResponse, error) {
	url := fmt.Sprintf("http://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		pickup.Longitude, pickup.Latitude,
		destination.Longitude, destination.Latitude,
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OSRM API: %v", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response: %v", err)
	}

	var routeRes tripTypes.OSRMApiResponse
	if err := json.Unmarshal(body, &routeRes); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &routeRes, nil
}

func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OSRMApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, fare := range baseFares {
		estimatedFares[i] = s.estimateFareRoute(fare, route)
	}

	return estimatedFares
}

func (s *service) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OSRMApiResponse) ([]*domain.RideFareModel, error) {
	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, fare := range rideFares {
		id := primitive.NewObjectID()

		fare = &domain.RideFareModel{
			ID:                id,
			UserID:            userID,
			TotalPriceInCents: fare.TotalPriceInCents,
			PackageSlug:       fare.PackageSlug,
			Route:             route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %v", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func (s *service) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip fare: %v", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("fare does not exist")
	}

	if fare.UserID != userID {
		return nil, fmt.Errorf("fare does not belong to the user")
	}

	return fare, nil
}

func (s *service) estimateFareRoute(fare *domain.RideFareModel, route *tripTypes.OSRMApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := fare.TotalPriceInCents

	distanceInKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceInKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricePerUnitOfTime

	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		PackageSlug:       fare.PackageSlug,
		TotalPriceInCents: totalPrice,
	}
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 1000,
		},
	}
}
