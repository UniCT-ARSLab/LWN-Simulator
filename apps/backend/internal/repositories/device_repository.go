package repositories

import (
	"lwn-simulator-backend/internal/database"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type DeviceRepository struct {
	repository *database.JSONRepository[contracts.Device]
}

func NewDeviceRepository(dataDir string) *DeviceRepository {
	return &DeviceRepository{
		repository: (*database.JSONRepository[contracts.Device])(database.NewJSONRepository[contracts.Device](
			dataDir,
			"devices.json",
		)),
	}
}

func deviceIdGetter(device contracts.Device) string {
	return device.ID
}

func (r *DeviceRepository) GetAll() ([]contracts.Device, error) {
	return r.repository.GetAll()
}

func (r *DeviceRepository) Save(devices []contracts.Device) error {
	return r.repository.Save(devices)
}

func (r *DeviceRepository) GetByID(id string) (contracts.Device, error) {
	return r.repository.GetByID(id, deviceIdGetter)
}

func (r *DeviceRepository) Update(device contracts.Device) error {
	return r.repository.Update(device.ID, device, deviceIdGetter)
}

func (r *DeviceRepository) Delete(device contracts.Device) error {
	return r.repository.Delete(device.ID, deviceIdGetter)
}
