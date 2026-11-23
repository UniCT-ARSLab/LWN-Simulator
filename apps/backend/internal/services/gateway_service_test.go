package services

import (
	"errors"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"lwn-simulator-backend/internal/database"
)

type mockGatewayRepository struct {
	gateways   []contracts.Gateway
	getAllErr  error
	getByIDErr error
	saveErr    error
	updateErr  error
	deleteErr  error
}

func (m *mockGatewayRepository) GetAll() ([]contracts.Gateway, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.gateways, nil
}

func (m *mockGatewayRepository) GetByID(id string) (contracts.Gateway, error) {
	if m.getByIDErr != nil {
		return contracts.Gateway{}, m.getByIDErr
	}
	for _, gateway := range m.gateways {
		if gateway.ID == id {
			return gateway, nil
		}
	}
	return contracts.Gateway{}, database.ErrNotFound
}

func (m *mockGatewayRepository) Save(gateways []contracts.Gateway) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.gateways = gateways
	return nil
}

func (m *mockGatewayRepository) Update(gateway contracts.Gateway) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i := range m.gateways {
		if m.gateways[i].ID == gateway.ID {
			m.gateways[i] = gateway
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *mockGatewayRepository) Delete(gateway contracts.Gateway) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i := range m.gateways {
		if m.gateways[i].ID == gateway.ID {
			m.gateways = append(m.gateways[:i], m.gateways[i+1:]...)
			return nil
		}
	}
	return database.ErrNotFound
}

func validCreateGatewayRequest() contracts.CreateGatewayRequest {
	keepAlive := int32(30)
	return contracts.CreateGatewayRequest{
		Name:       "Virtual gateway",
		Type:       contracts.Virtual,
		GatewayEUI: "0102030405060708",
		KeepAlive:  &keepAlive,
		Latitude:   float32Pointer(37.5),
		Longitude:  float32Pointer(15),
		Altitude:   float32Pointer(100),
	}
}

func TestGatewayService_CreateGateway(t *testing.T) {
	tests := []struct {
		name          string
		repository    *mockGatewayRepository
		request       contracts.CreateGatewayRequest
		wantErr       bool
		errorContains string
	}{
		{
			name:       "creates gateway active by default",
			repository: &mockGatewayRepository{},
			request:    validCreateGatewayRequest(),
		},
		{
			name: "rejects duplicate GatewayEUI",
			repository: &mockGatewayRepository{gateways: []contracts.Gateway{{
				GatewayEUI: "0102030405060708",
			}}},
			request:       validCreateGatewayRequest(),
			wantErr:       true,
			errorContains: "already exists",
		},
		{
			name: "returns repository read errors",
			repository: &mockGatewayRepository{
				getAllErr: errors.New("database error"),
			},
			request:       validCreateGatewayRequest(),
			wantErr:       true,
			errorContains: "get gateways",
		},
		{
			name:          "returns repository write errors",
			repository:    &mockGatewayRepository{saveErr: errors.New("database error")},
			request:       validCreateGatewayRequest(),
			wantErr:       true,
			errorContains: "save gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGatewayService(tt.repository)
			gateway, err := service.CreateGateway(tt.request)

			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateGateway() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("CreateGateway() error = %q, want substring %q", err, tt.errorContains)
			}
			if !tt.wantErr && !gateway.Active {
				t.Error("CreateGateway() should set active=true")
			}
		})
	}
}

func float32Pointer(value float32) *float32 { return &value }

func TestValidateGatewayRequest(t *testing.T) {
	tests := []struct {
		name    string
		request contracts.CreateGatewayRequest
		valid   bool
	}{
		{name: "virtual gateway", request: validCreateGatewayRequest(), valid: true},
		{
			name: "real gateway",
			request: func() contracts.CreateGatewayRequest {
				request := validCreateGatewayRequest()
				port := int32(1700)
				ip := "192.168.1.20"
				request.Type = contracts.Real
				request.KeepAlive = nil
				request.GatewayIPv4 = &ip
				request.GatewayPort = &port
				return request
			}(),
			valid: true,
		},
		{
			name: "virtual gateway requires keepAlive",
			request: func() contracts.CreateGatewayRequest {
				request := validCreateGatewayRequest()
				request.KeepAlive = nil
				return request
			}(),
		},
		{
			name: "real gateway requires IPv4 and port",
			request: func() contracts.CreateGatewayRequest {
				request := validCreateGatewayRequest()
				request.Type = contracts.Real
				request.KeepAlive = nil
				return request
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGatewayRequest(tt.request); (err == nil) != tt.valid {
				t.Fatalf("ValidateGatewayRequest() error = %v, valid = %v", err, tt.valid)
			}
		})
	}
}
