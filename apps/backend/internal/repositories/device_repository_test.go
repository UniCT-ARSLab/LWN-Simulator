package repositories

import (
	"reflect"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

const (
	testDeviceID1 = "550e8400-e29b-41d4-a716-446655440000"
	testDeviceID2 = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testDeviceID3 = "7296a689-84e6-432c-826c-4b703d550ac0"
)

func testDevices() []contracts.Device {
	return []contracts.Device{
		{
			ID:     testDeviceID1,
			DevEUI: "0102030405060708",
			Name:   "device-1",
		},
		{
			ID:     testDeviceID2,
			DevEUI: "1122334455667788",
			Name:   "device-2",
		},
		{
			ID:     testDeviceID3,
			DevEUI: "AABBCCDDEEFF0011",
			Name:   "device-3",
		},
	}
}

func newTestDeviceRepository(t *testing.T) *DeviceRepository {
	t.Helper()

	return NewDeviceRepository(t.TempDir())
}

func TestDeviceRepository_GetAll(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *DeviceRepository)
		want  []contracts.Device
	}{
		{
			name: "returns all devices",
			setup: func(t *testing.T, repository *DeviceRepository) {
				t.Helper()

				devices := testDevices()

				if err := repository.Save(devices); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			},
			want: testDevices(),
		},
		{
			name: "returns empty slice when database is empty",
			setup: func(t *testing.T, repository *DeviceRepository) {
				t.Helper()

				if err := repository.Save([]contracts.Device{}); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			},
			want: []contracts.Device{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestDeviceRepository(t)

			tt.setup(t, repository)

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"GetAll() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestDeviceRepository_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		want    contracts.Device
		wantErr bool
	}{
		{
			name: "returns existing device",
			id:   testDeviceID2,
			want: contracts.Device{
				ID:     testDeviceID2,
				DevEUI: "1122334455667788",
				Name:   "device-2",
			},
		},
		{
			name:    "returns error when device does not exist",
			id:      "unknown-device",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestDeviceRepository(t)

			if err := repository.Save(testDevices()); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			got, err := repository.GetByID(tt.id)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"GetByID() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"GetByID() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestDeviceRepository_Save(t *testing.T) {
	tests := []struct {
		name    string
		devices []contracts.Device
	}{
		{
			name: "saves multiple devices",
			devices: []contracts.Device{
				{
					ID:     testDeviceID1,
					DevEUI: "0102030405060708",
					Name:   "device-1",
				},
				{
					ID:     testDeviceID2,
					DevEUI: "1122334455667788",
					Name:   "device-2",
				},
			},
		},
		{
			name:    "saves empty slice",
			devices: []contracts.Device{},
		},
		{
			name:    "saves nil slice",
			devices: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestDeviceRepository(t)

			if err := repository.Save(tt.devices); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.devices) {
				t.Errorf(
					"GetAll() = %v, want %v",
					got,
					tt.devices,
				)
			}
		})
	}
}

func TestDeviceRepository_Update(t *testing.T) {
	tests := []struct {
		name    string
		update  contracts.Device
		want    []contracts.Device
		wantErr bool
	}{
		{
			name: "updates existing device",
			update: contracts.Device{
				ID:     testDeviceID2,
				DevEUI: "FFEEDDCCBBAA9988",
				Name:   "updated-device",
				Active: true,
			},
			want: []contracts.Device{
				{
					ID:     testDeviceID1,
					DevEUI: "0102030405060708",
					Name:   "device-1",
				},
				{
					ID:     testDeviceID2,
					DevEUI: "FFEEDDCCBBAA9988",
					Name:   "updated-device",
					Active: true,
				},
				{
					ID:     testDeviceID3,
					DevEUI: "AABBCCDDEEFF0011",
					Name:   "device-3",
				},
			},
		},
		{
			name: "returns error when device does not exist",
			update: contracts.Device{
				ID:     "unknown-device",
				DevEUI: "0102030405060708",
				Name:   "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestDeviceRepository(t)

			initial := testDevices()

			if err := repository.Save(initial); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			err := repository.Update(tt.update)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"Update() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				return
			}

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"devices after Update() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestDeviceRepository_Delete(t *testing.T) {
	tests := []struct {
		name    string
		device  contracts.Device
		want    []contracts.Device
		wantErr bool
	}{
		{
			name: "deletes existing device",
			device: contracts.Device{
				ID: testDeviceID2,
			},
			want: []contracts.Device{
				{
					ID:     testDeviceID1,
					DevEUI: "0102030405060708",
					Name:   "device-1",
				},
				{
					ID:     testDeviceID3,
					DevEUI: "AABBCCDDEEFF0011",
					Name:   "device-3",
				},
			},
		},
		{
			name: "deletes first device",
			device: contracts.Device{
				ID: testDeviceID1,
			},
			want: []contracts.Device{
				{
					ID:     testDeviceID2,
					DevEUI: "1122334455667788",
					Name:   "device-2",
				},
				{
					ID:     testDeviceID3,
					DevEUI: "AABBCCDDEEFF0011",
					Name:   "device-3",
				},
			},
		},
		{
			name: "deletes last device",
			device: contracts.Device{
				ID: testDeviceID3,
			},
			want: []contracts.Device{
				{
					ID:     testDeviceID1,
					DevEUI: "0102030405060708",
					Name:   "device-1",
				},
				{
					ID:     testDeviceID2,
					DevEUI: "1122334455667788",
					Name:   "device-2",
				},
			},
		},
		{
			name: "returns error when device does not exist",
			device: contracts.Device{
				ID: "unknown-device",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestDeviceRepository(t)

			if err := repository.Save(testDevices()); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			err := repository.Delete(tt.device)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"Delete() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				return
			}

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"devices after Delete() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestDeviceRepository_DeviceIDGetter(t *testing.T) {
	tests := []struct {
		name   string
		device contracts.Device
		want   string
	}{
		{
			name: "returns device ID",
			device: contracts.Device{
				ID: "device-123",
			},
			want: "device-123",
		},
		{
			name:   "returns empty ID",
			device: contracts.Device{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deviceIdGetter(tt.device)

			if got != tt.want {
				t.Errorf(
					"deviceIdGetter() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
