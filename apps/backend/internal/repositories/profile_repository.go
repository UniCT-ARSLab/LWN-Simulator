package repositories

import (
	"lwn-simulator-backend/internal/database"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type ProfileRepository struct {
	repository *database.JSONRepository[contracts.Profile]
}

func NewProfileRepository(dataDir string) *ProfileRepository {
	return &ProfileRepository{
		repository: (*database.JSONRepository[contracts.Profile])(database.NewJSONRepository[contracts.Profile](dataDir, "profiles.json")),
	}
}

func profileIDGetter(profile contracts.Profile) string {
	return profile.ID
}

func (r *ProfileRepository) GetAll() ([]contracts.Profile, error) {
	return r.repository.GetAll()
}

func (r *ProfileRepository) GetByID(id string) (contracts.Profile, error) {
	return r.repository.GetByID(id, profileIDGetter)
}

func (r *ProfileRepository) Save(profiles []contracts.Profile) error {
	return r.repository.Save(profiles)
}

func (r *ProfileRepository) Update(profile contracts.Profile) error {
	return r.repository.Update(profile.ID, profile, profileIDGetter)
}

func (r *ProfileRepository) Delete(profile contracts.Profile) error {
	return r.repository.Delete(profile.ID, profileIDGetter)
}
