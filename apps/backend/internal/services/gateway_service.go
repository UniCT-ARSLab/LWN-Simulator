package services

import (
	"errors"
	"fmt"
	"sync"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/google/uuid"
	"lwn-simulator-backend/internal/apperrors"
	"lwn-simulator-backend/internal/database"
)

type GatewayRepository interface {
	GetAll() ([]contracts.Gateway, error)
	GetByID(id string) (contracts.Gateway, error)
	Save(gateways []contracts.Gateway) error
	Update(gateway contracts.Gateway) error
	Delete(gateway contracts.Gateway) error
}

type GatewayRuntimeSynchronizer interface {
	AcquireHardwareMutation() func()
	RegisterGatewayLocked(contracts.Gateway) error
	ValidateGatewayUpdateLocked(contracts.Gateway) error
	UpdateGatewayLocked(contracts.Gateway) error
	RemoveGatewayLocked(string) error
}

type GatewayService struct {
	repository GatewayRepository
	runtime    GatewayRuntimeSynchronizer
	realtime   RealtimePublisher
	mu         sync.Mutex
}

func (s *GatewayService) SetRealtimePublisher(publisher RealtimePublisher) {
	s.mu.Lock()
	s.realtime = publisher
	s.mu.Unlock()
}

func NewGatewayService(repository GatewayRepository) *GatewayService {
	return &GatewayService{repository: repository}
}

func (s *GatewayService) SetRuntimeSynchronizer(runtime GatewayRuntimeSynchronizer) {
	s.mu.Lock()
	s.runtime = runtime
	s.mu.Unlock()
}

func (s *GatewayService) CreateGateway(req contracts.CreateGatewayRequest) (contracts.Gateway, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	if err := ValidateGatewayRequest(req); err != nil {
		return contracts.Gateway{}, err
	}

	gateways, err := s.repository.GetAll()
	if err != nil {
		return contracts.Gateway{}, fmt.Errorf("get gateways: %w", err)
	}

	for _, gateway := range gateways {
		if gateway.GatewayEUI == req.GatewayEUI {
			return contracts.Gateway{}, apperrors.Conflict(
				"gateway with gatewayEUI %q already exists",
				req.GatewayEUI,
			)
		}
	}

	gateway := contracts.Gateway{
		ID:          uuid.NewString(),
		Active:      true,
		Name:        req.Name,
		Type:        req.Type,
		GatewayEUI:  req.GatewayEUI,
		KeepAlive:   req.KeepAlive,
		GatewayIPv4: req.GatewayIPv4,
		GatewayPort: req.GatewayPort,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Altitude:    req.Altitude,
	}

	gateways = append(gateways, gateway)
	if err := s.repository.Save(gateways); err != nil {
		return contracts.Gateway{}, fmt.Errorf("save gateway: %w", err)
	}
	if s.runtime != nil {
		if err := s.runtime.RegisterGatewayLocked(gateway); err != nil {
			rollbackErr := s.repository.Delete(gateway)
			return contracts.Gateway{}, formatRollbackError("synchronize created gateway", err, rollbackErr)
		}
	}
	s.publishRealtime(contracts.RealtimeGatewayCreatedMessage, &gateway, gateway.ID)

	return gateway, nil
}

func (s *GatewayService) GetGateway(req contracts.GetGatewayRequest) (contracts.GetGatewayResponse, error) {
	gateway, err := s.repository.GetByID(req.ID)
	if err != nil {
		return contracts.GetGatewayResponse{}, wrapGatewayRepositoryError("get gateway by id", err)
	}
	return contracts.GetGatewayResponse{Gateway: gateway}, nil
}

func (s *GatewayService) GetGateways(req contracts.GetGatewaysRequest) (contracts.GetGatewaysResponse, error) {
	gateways, err := s.repository.GetAll()
	if err != nil {
		return contracts.GetGatewaysResponse{}, fmt.Errorf("get all gateways: %w", err)
	}
	if gateways == nil {
		gateways = []contracts.Gateway{}
	}
	return contracts.GetGatewaysResponse{Gateways: gateways}, nil
}

// ImportGateways replaces the gateways in a newly created profile while
// preserving their IDs for historical event references.
func (s *GatewayService) ImportGateways(gateways []contracts.Gateway) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	seenIDs := make(map[string]struct{}, len(gateways))
	seenEUIs := make(map[string]struct{}, len(gateways))
	for _, gateway := range gateways {
		if _, err := uuid.Parse(gateway.ID); err != nil {
			return apperrors.Invalid("gateway %q has an invalid id", gateway.ID)
		}
		if err := ValidateGateway(gateway); err != nil {
			return err
		}
		if _, exists := seenIDs[gateway.ID]; exists {
			return apperrors.Conflict("duplicate gateway id %q", gateway.ID)
		}
		if _, exists := seenEUIs[gateway.GatewayEUI]; exists {
			return apperrors.Conflict("duplicate gateway EUI %q", gateway.GatewayEUI)
		}
		seenIDs[gateway.ID] = struct{}{}
		seenEUIs[gateway.GatewayEUI] = struct{}{}
	}

	if err := s.repository.Save(gateways); err != nil {
		return fmt.Errorf("import gateways: %w", err)
	}
	if s.runtime != nil {
		for _, gateway := range gateways {
			if err := s.runtime.RegisterGatewayLocked(gateway); err != nil {
				return fmt.Errorf("synchronize imported gateway %q: %w", gateway.ID, err)
			}
		}
	}
	return nil
}

func (s *GatewayService) UpdateGateway(req contracts.UpdateGatewayRequest) (contracts.UpdateGatewayResponse, error) {
	if err := ValidateGateway(req.Gateway); err != nil {
		return contracts.UpdateGatewayResponse{}, err
	}
	return s.updateGateway(req.ID, func(gateway *contracts.Gateway) error {
		*gateway = req.Gateway
		return nil
	})
}

func (s *GatewayService) updateGateway(
	id string,
	apply func(*contracts.Gateway) error,
) (contracts.UpdateGatewayResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	previous, err := s.repository.GetByID(id)
	if err != nil {
		return contracts.UpdateGatewayResponse{}, wrapGatewayRepositoryError("get gateway by id", err)
	}

	gateway := previous
	if err := apply(&gateway); err != nil {
		return contracts.UpdateGatewayResponse{}, err
	}
	gateway.ID = id

	gateways, err := s.repository.GetAll()
	if err != nil {
		return contracts.UpdateGatewayResponse{}, fmt.Errorf("get gateways for update: %w", err)
	}

	for _, existing := range gateways {
		if existing.ID == id {
			continue
		}
		if existing.GatewayEUI == gateway.GatewayEUI {
			return contracts.UpdateGatewayResponse{}, apperrors.Conflict(
				"gateway with gatewayEUI %q already exists",
				gateway.GatewayEUI,
			)
		}
	}

	if s.runtime != nil {
		if err := s.runtime.ValidateGatewayUpdateLocked(gateway); err != nil {
			return contracts.UpdateGatewayResponse{}, err
		}
	}
	if err := s.repository.Update(gateway); err != nil {
		return contracts.UpdateGatewayResponse{}, wrapGatewayRepositoryError("update gateway", err)
	}
	if s.runtime != nil {
		if err := s.runtime.UpdateGatewayLocked(gateway); err != nil {
			rollbackErr := s.repository.Update(previous)
			return contracts.UpdateGatewayResponse{}, formatRollbackError("synchronize updated gateway", err, rollbackErr)
		}
	}
	s.publishRealtime(contracts.RealtimeGatewayUpdatedMessage, &gateway, gateway.ID)

	return contracts.UpdateGatewayResponse{Gateway: gateway}, nil
}

func (s *GatewayService) DeleteGateway(req contracts.DeleteGatewayRequest) (contracts.DeleteGatewayResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	gateway, err := s.repository.GetByID(req.ID)
	if err != nil {
		return contracts.DeleteGatewayResponse{}, wrapGatewayRepositoryError("get gateway by id", err)
	}
	if err := s.repository.Delete(gateway); err != nil {
		return contracts.DeleteGatewayResponse{}, wrapGatewayRepositoryError("delete gateway", err)
	}
	if s.runtime != nil {
		if err := s.runtime.RemoveGatewayLocked(gateway.ID); err != nil {
			gateways, readErr := s.repository.GetAll()
			if readErr != nil {
				return contracts.DeleteGatewayResponse{}, formatRollbackError("synchronize deleted gateway", err, readErr)
			}
			gateways = append(gateways, gateway)
			rollbackErr := s.repository.Save(gateways)
			return contracts.DeleteGatewayResponse{}, formatRollbackError("synchronize deleted gateway", err, rollbackErr)
		}
	}
	s.publishRealtime(contracts.RealtimeGatewayDeletedMessage, nil, gateway.ID)
	return contracts.DeleteGatewayResponse{}, nil
}

func (s *GatewayService) publishRealtime(
	messageType contracts.RealtimeWebSocketMessageType,
	gateway *contracts.Gateway,
	resourceID string,
) {
	publisher := s.realtime
	if publisher == nil {
		return
	}
	publisher.Publish(contracts.RealtimeWebSocketMessage{
		Type:                  messageType,
		TimestampMilliseconds: 0,
		ResourceID:            stringPointer(resourceID),
		Gateway:               gateway,
	})
}

func wrapGatewayRepositoryError(operation string, err error) error {
	if errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("%w: %s: %v", apperrors.ErrNotFound, operation, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
