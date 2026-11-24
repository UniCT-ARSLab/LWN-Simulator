package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"lwn-simulator-backend/internal/database"
	"lwn-simulator-backend/internal/repositories"
)

func newTestProfileService(t *testing.T) (*ProfileService, string) {
	t.Helper()
	dataDir := t.TempDir()
	return NewProfileService(repositories.NewProfileRepository(dataDir), dataDir), dataDir
}

func TestProfileServiceEnsuresStableDefaultProfile(t *testing.T) {
	service, dataDir := newTestProfileService(t)
	first, err := service.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.ID != database.DefaultProfileID {
		t.Fatalf("default profile is not stable: first=%+v second=%+v", first, second)
	}
	storage, err := database.NewProfileStorage(dataDir, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storage.Directory()); err != nil {
		t.Fatalf("default profile storage was not created: %v", err)
	}
}

func TestProfileServiceCreateRenameAndDelete(t *testing.T) {
	service, dataDir := newTestProfileService(t)
	if _, err := service.EnsureDefaultProfile(); err != nil {
		t.Fatal(err)
	}
	created, err := service.Create("  Field test  ")
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Field test" || created.ID == database.DefaultProfileID {
		t.Fatalf("unexpected created profile: %+v", created)
	}
	if _, err := service.Create("field TEST"); err == nil {
		t.Fatal("duplicate profile name was accepted")
	}
	renamed, err := service.Rename(created.ID, "Production")
	if err != nil || renamed.Name != "Production" {
		t.Fatalf("Rename() = %+v, err=%v", renamed, err)
	}
	storage, err := database.NewProfileStorage(dataDir, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(storage.Directory(), "marker.json")
	if err := os.WriteFile(marker, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storage.Directory()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile storage was not removed: err=%v", err)
	}
}

func TestProfileServiceCannotDeleteLastProfile(t *testing.T) {
	service, _ := newTestProfileService(t)
	if _, err := service.EnsureDefaultProfile(); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(database.DefaultProfileID); !errors.Is(err, ErrCannotDeleteLastProfile) {
		t.Fatalf("Delete() error = %v, want ErrCannotDeleteLastProfile", err)
	}
}

func TestProfileServiceRejectsInvalidNames(t *testing.T) {
	service, _ := newTestProfileService(t)
	if _, err := service.EnsureDefaultProfile(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "   ", string(make([]rune, 101))} {
		if _, err := service.Create(name); err == nil {
			t.Fatalf("invalid name %q was accepted", name)
		}
	}
}

var _ contracts.Profile

func TestFailedProfilePopulationIsNeverPublished(t *testing.T) {
	service, dir := newTestProfileService(t)
	if _, err := service.EnsureDefaultProfile(); err != nil {
		t.Fatal(err)
	}
	publisher := &recordingRealtimePublisher{messages: make(chan contracts.RealtimeWebSocketMessage, 2)}
	service.SetRealtimePublisher(publisher)
	failure := errors.New("archive write failed")
	var partial string
	_, err := service.create("Import", func(storage database.ProfileStorage) error {
		partial = storage.Directory()
		all, err := repositories.NewProfileRepository(dir).GetAll()
		if err != nil || len(all) != 1 {
			t.Fatalf("visible incomplete profile: %v %v", all, err)
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("unexpected import error: %v", err)
	}
	if _, err := os.Stat(partial); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial storage: %v", err)
	}
	if len(publisher.messages) != 0 {
		t.Fatal("notified clients of failed import")
	}
}
