package repositories

import (
	"reflect"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestProfileRepositoryCRUD(t *testing.T) {
	repository := NewProfileRepository(t.TempDir())
	profiles := []contracts.Profile{
		{ID: "00000000-0000-4000-8000-000000000001", Name: "Default profile"},
		{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Test network"},
	}
	if err := repository.Save(profiles); err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, profiles) {
		t.Fatalf("profiles = %+v, want %+v", loaded, profiles)
	}
	profile, err := repository.GetByID(profiles[1].ID)
	if err != nil || profile != profiles[1] {
		t.Fatalf("GetByID() = %+v, err=%v", profile, err)
	}
	profiles[1].Name = "Renamed network"
	if err := repository.Update(profiles[1]); err != nil {
		t.Fatal(err)
	}
	if err := repository.Delete(profiles[0]); err != nil {
		t.Fatal(err)
	}
	loaded, err = repository.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Name != "Renamed network" {
		t.Fatalf("unexpected profiles after update/delete: %+v", loaded)
	}
}
