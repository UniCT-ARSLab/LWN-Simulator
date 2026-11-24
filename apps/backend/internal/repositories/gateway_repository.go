package repositories

import (
	"lwn-simulator-backend/internal/database"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type GatewayRepository struct {
	repository *database.JSONRepository[contracts.Gateway]
}

func NewGatewayRepository(dataDir string) *GatewayRepository {
	return &GatewayRepository{
		repository: (*database.JSONRepository[contracts.Gateway])(database.NewJSONRepository[contracts.Gateway](
			dataDir,
			"gateways.json",
		)),
	}
}

func (r *GatewayRepository) GetAll() ([]contracts.Gateway, error) {
	return r.repository.GetAll()
}

func (r *GatewayRepository) Save(gateways []contracts.Gateway) error {
	return r.repository.Save(gateways)
}

func gatewayIDGetter(gateway contracts.Gateway) string {
	return gateway.ID
}

func (r *GatewayRepository) GetByID(id string) (contracts.Gateway, error) {
	return r.repository.GetByID(id, gatewayIDGetter)
}

func (r *GatewayRepository) Update(gateway contracts.Gateway) error {
	return r.repository.Update(gateway.ID, gateway, gatewayIDGetter)
}

func (r *GatewayRepository) Delete(gateway contracts.Gateway) error {
	return r.repository.Delete(gateway.ID, gatewayIDGetter)
}
