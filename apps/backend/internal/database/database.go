package database

import (
	"embed"
	"errors"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/google/uuid"
)

const applicationName = "lwn-simulator"

const (
	dataDirPermissions  = 0700
	dataFilePermissions = 0600
	profilesDirectory   = "profiles"
)

// DefaultProfileID is stable across installations so existing data can be
// migrated without changing the identity of the default network.
const DefaultProfileID = "00000000-0000-4000-8000-000000000001"

// ProfileStorage describes the filesystem location owned by one profile.
// Keeping this path calculation in the database package prevents individual
// repositories from accidentally sharing data between profiles.
type ProfileStorage struct {
	profileID string
	directory string
}

//go:embed defaults/*.json
var defaults embed.FS

func DataDir() (string, error) {
	if os.Getenv("LWN_ENV") == "development" {
		return filepath.Abs("./dev-db")
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}

		return filepath.Join(appData, applicationName), nil

	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}

		return filepath.Join(
			home,
			"Library",
			"Application Support",
			applicationName,
		), nil

	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}

		return filepath.Join(
			home,
			".local",
			"share",
			applicationName,
		), nil
	}
}

func Initialize() (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dataDir, dataDirPermissions); err != nil {
		return "", fmt.Errorf("create data directory: %w", err)
	}

	if err := initializeFile(dataDir, "profiles.json"); err != nil {
		return "", err
	}
	profiles, err := NewJSONRepository[contracts.Profile](dataDir, "profiles.json").GetAll()
	if err != nil {
		return "", err
	}
	if len(profiles) > 0 && !slices.ContainsFunc(profiles, func(profile contracts.Profile) bool { return profile.ID == DefaultProfileID }) {
		// The user deleted the default profile after migrating to named profiles.
		return dataDir, nil
	}
	for _, name := range []string{"devices.json", "gateways.json"} {
		if err := initializeFile(dataDir, name); err != nil {
			return "", err
		}
	}
	if err := migrateLegacyData(dataDir); err != nil {
		return "", fmt.Errorf("migrate legacy data to default profile: %w", err)
	}

	return dataDir, nil
}

// NewProfileStorage returns the isolated storage location for a profile.
func NewProfileStorage(dataDir string, profileID string) (ProfileStorage, error) {
	if strings.TrimSpace(dataDir) == "" {
		return ProfileStorage{}, fmt.Errorf("data directory is required")
	}
	if _, err := uuid.Parse(profileID); err != nil {
		return ProfileStorage{}, fmt.Errorf("invalid profile id %q: %w", profileID, err)
	}

	directory := filepath.Join(dataDir, profilesDirectory, profileID)

	return ProfileStorage{profileID: profileID, directory: directory}, nil
}

// ProfileID returns the profile owning the storage location.
func (storage ProfileStorage) ProfileID() string {
	return storage.profileID
}

// Directory returns the profile's data directory.
func (storage ProfileStorage) Directory() string {
	return storage.directory
}

// Ensure creates the profile directory with application-private permissions.
func (storage ProfileStorage) Ensure() error {
	if err := os.MkdirAll(storage.directory, dataDirPermissions); err != nil {
		return fmt.Errorf("create profile data directory: %w", err)
	}
	return nil
}

// EnsureDataFiles creates the empty collections required by a new profile.
// Existing files are never modified, which makes this safe after migration.
func (storage ProfileStorage) EnsureDataFiles() error {
	if err := storage.Ensure(); err != nil {
		return err
	}
	initialContents := map[string]string{
		"devices.json":         "[]\n",
		"gateways.json":        "[]\n",
		"simulation-logs.json": "[]\n",
		"gateway-bridge.json":  "{\"enabled\":false,\"address\":\"0.0.0.0\",\"port\":1700}\n",
	}
	for name, contents := range initialContents {
		path, err := storage.File(name)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, dataFilePermissions)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("create profile data file %s: %w", name, err)
		}
		if _, err := file.WriteString(contents); err != nil {
			_ = file.Close()
			return fmt.Errorf("initialize profile data file %s: %w", name, err)
		}
		if err := file.Sync(); err != nil {
			_ = file.Close()
			return fmt.Errorf("sync profile data file %s: %w", name, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close profile data file %s: %w", name, err)
		}
	}
	return nil
}

// File returns a path for a profile-owned file. File names are deliberately
// restricted to a single path component to prevent traversal outside the
// profile directory.
func (storage ProfileStorage) File(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", fmt.Errorf("invalid profile file name %q", name)
	}
	return filepath.Join(storage.directory, name), nil
}

var legacyProfileFiles = []string{
	"devices.json",
	"gateways.json",
	"simulation-logs.json",
	"simulation-checkpoint.json",
}

// migrateLegacyData moves the pre-profile files into the default profile.
// Rename is used instead of copying so a restart cannot leave two active
// sources of truth. The operation is idempotent and never overwrites a file
// that has already been migrated.
func migrateLegacyData(dataDir string) error {
	storage, err := NewProfileStorage(dataDir, DefaultProfileID)
	if err != nil {
		return err
	}
	if err := storage.Ensure(); err != nil {
		return err
	}

	for _, name := range legacyProfileFiles {
		source := filepath.Join(dataDir, name)
		destination, err := storage.File(name)
		if err != nil {
			return err
		}
		if source == destination {
			continue
		}
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("inspect legacy file %s: %w", name, err)
		}
		if _, err := os.Stat(destination); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect migrated file %s: %w", name, err)
		}
		if err := os.Rename(source, destination); err != nil {
			return fmt.Errorf("move %s: %w", name, err)
		}
	}

	legacyEvents := filepath.Join(dataDir, "simulation-logs.events")
	destinationEvents := filepath.Join(storage.Directory(), "simulation-logs.events")
	if _, err := os.Stat(legacyEvents); err == nil {
		if _, destinationErr := os.Stat(destinationEvents); os.IsNotExist(destinationErr) {
			if err := os.Rename(legacyEvents, destinationEvents); err != nil {
				return fmt.Errorf("move simulation event sidecar: %w", err)
			}
		} else if destinationErr != nil {
			return fmt.Errorf("inspect migrated simulation event sidecar: %w", destinationErr)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect legacy simulation event sidecar: %w", err)
	}

	return storage.EnsureDataFiles()
}

func initializeFile(dataDir string, name string) error {
	path := filepath.Join(dataDir, name)

	if _, err := os.Stat(path); err == nil {
		if err := os.Chmod(path, dataFilePermissions); err != nil {
			return fmt.Errorf("set permissions for database file %s: %w", name, err)
		}

		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check database file %s: %w", name, err)
	}

	data, err := defaults.ReadFile(pathpkg.Join("defaults", name))
	if err != nil {
		return fmt.Errorf("read embedded default %s: %w", name, err)
	}

	if err := os.WriteFile(path, data, dataFilePermissions); err != nil {
		return fmt.Errorf("create database file %s: %w", name, err)
	}

	return nil
}
