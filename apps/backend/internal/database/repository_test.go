package database

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func testItemID(item testItem) string {
	return item.ID
}

func newTestRepository(t *testing.T) (*JSONRepository[testItem], string) {
	t.Helper()

	dataDir := t.TempDir()

	repository := NewJSONRepository[testItem](
		dataDir,
		"items.json",
	)

	return repository, filepath.Join(dataDir, "items.json")
}

func writeTestFile(t *testing.T, path string, data any) {
	t.Helper()

	content, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func TestJSONRepository_GetAll(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		want        []testItem
		wantErr     bool
	}{
		{
			name: "returns all items",
			fileContent: `[
				{"id":"1","name":"first"},
				{"id":"2","name":"second"}
			]`,
			want: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
		},
		{
			name:        "returns empty slice for empty file",
			fileContent: "",
			want:        []testItem{},
		},
		{
			name:        "returns error for invalid json",
			fileContent: `{invalid json}`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, path := newTestRepository(t)

			if err := os.WriteFile(
				path,
				[]byte(tt.fileContent),
				0644,
			); err != nil {
				t.Fatalf("write test file: %v", err)
			}

			got, err := repository.GetAll()

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetAll() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONRepository_GetAll_FileNotFound(t *testing.T) {
	repository, _ := newTestRepository(t)

	_, err := repository.GetAll()

	if err == nil {
		t.Fatal("GetAll() expected error, got nil")
	}
}

func TestJSONRepository_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		items      []testItem
		want       testItem
		wantErr    bool
		errorMatch string
	}{
		{
			name: "returns existing item",
			id:   "2",
			items: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			want: testItem{
				ID:   "2",
				Name: "second",
			},
		},
		{
			name: "returns error when item does not exist",
			id:   "999",
			items: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			wantErr:    true,
			errorMatch: `item with id "999" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, path := newTestRepository(t)

			writeTestFile(t, path, tt.items)

			got, err := repository.GetByID(
				tt.id,
				testItemID,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"GetByID() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errorMatch) {
					t.Errorf(
						"GetByID() error = %q, want substring %q",
						err,
						tt.errorMatch,
					)
				}

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONRepository_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		item       testItem
		initial    []testItem
		want       []testItem
		wantErr    bool
		errorMatch string
	}{
		{
			name: "updates existing item",
			id:   "2",
			item: testItem{
				ID:   "2",
				Name: "updated",
			},
			initial: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			want: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "updated"},
			},
		},
		{
			name: "returns error when item does not exist",
			id:   "999",
			item: testItem{
				ID:   "999",
				Name: "new",
			},
			initial: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			wantErr:    true,
			errorMatch: `item with id "999" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, path := newTestRepository(t)

			writeTestFile(t, path, tt.initial)

			err := repository.Update(
				tt.id,
				tt.item,
				testItemID,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"Update() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errorMatch) {
					t.Errorf(
						"Update() error = %q, want substring %q",
						err,
						tt.errorMatch,
					)
				}

				return
			}

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() after Update() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"items after Update() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestJSONRepository_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		initial    []testItem
		want       []testItem
		wantErr    bool
		errorMatch string
	}{
		{
			name: "deletes existing item",
			id:   "2",
			initial: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
				{ID: "3", Name: "third"},
			},
			want: []testItem{
				{ID: "1", Name: "first"},
				{ID: "3", Name: "third"},
			},
		},
		{
			name: "deletes first item",
			id:   "1",
			initial: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			want: []testItem{
				{ID: "2", Name: "second"},
			},
		},
		{
			name: "deletes last item",
			id:   "2",
			initial: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			want: []testItem{
				{ID: "1", Name: "first"},
			},
		},
		{
			name: "returns error when item does not exist",
			id:   "999",
			initial: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
			wantErr:    true,
			errorMatch: `item with id "999" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, path := newTestRepository(t)

			writeTestFile(t, path, tt.initial)

			err := repository.Delete(
				tt.id,
				testItemID,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"Delete() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errorMatch) {
					t.Errorf(
						"Delete() error = %q, want substring %q",
						err,
						tt.errorMatch,
					)
				}

				return
			}

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() after Delete() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"items after Delete() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestJSONRepository_Save(t *testing.T) {
	tests := []struct {
		name  string
		items []testItem
	}{
		{
			name: "saves multiple items",
			items: []testItem{
				{ID: "1", Name: "first"},
				{ID: "2", Name: "second"},
			},
		},
		{
			name:  "saves empty slice",
			items: []testItem{},
		},
		{
			name:  "saves nil slice",
			items: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, path := newTestRepository(t)

			if err := repository.Save(tt.items); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			got, err := repository.GetAll()
			if err != nil {
				t.Fatalf("GetAll() after Save() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.items) {
				t.Errorf(
					"items after Save() = %v, want %v",
					got,
					tt.items,
				)
			}

			if _, err := os.Stat(path); err != nil {
				t.Errorf("Save() did not create %s: %v", path, err)
			}
		})
	}
}

func TestJSONRepository_Save_CreatesDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "nested", "database")

	repository := NewJSONRepository[testItem](
		dataDir,
		"items.json",
	)

	items := []testItem{
		{ID: "1", Name: "first"},
	}

	if err := repository.Save(items); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repository.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if !reflect.DeepEqual(got, items) {
		t.Errorf("GetAll() = %v, want %v", got, items)
	}
}

func TestJSONRepository_Save_OverwritesExistingData(t *testing.T) {
	repository, _ := newTestRepository(t)

	initial := []testItem{
		{ID: "1", Name: "first"},
	}

	if err := repository.Save(initial); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	replacement := []testItem{
		{ID: "2", Name: "second"},
	}

	if err := repository.Save(replacement); err != nil {
		t.Fatalf("replacement Save() error = %v", err)
	}

	got, err := repository.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if !reflect.DeepEqual(got, replacement) {
		t.Errorf("GetAll() = %v, want %v", got, replacement)
	}
}

func TestJSONRepository_Operations_ReturnLoadError(t *testing.T) {
	tests := []struct {
		name string
		run  func(*JSONRepository[testItem]) error
	}{
		{
			name: "Update",
			run: func(r *JSONRepository[testItem]) error {
				return r.Update(
					"1",
					testItem{ID: "1", Name: "updated"},
					testItemID,
				)
			},
		},
		{
			name: "Delete",
			run: func(r *JSONRepository[testItem]) error {
				return r.Delete("1", testItemID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, _ := newTestRepository(t)

			err := tt.run(repository)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), "read") {
				t.Errorf(
					"error = %q, expected read error",
					err,
				)
			}
		})
	}
}

func TestJSONRepository_GetByID_ReturnsLoadError(t *testing.T) {
	repository, _ := newTestRepository(t)

	_, err := repository.GetByID("1", testItemID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "read") {
		t.Errorf(
			"error = %q, expected read error",
			err,
		)
	}
}

func TestJSONRepository_ConcurrentAccess(t *testing.T) {
	repository, _ := newTestRepository(t)

	initial := []testItem{
		{ID: "1", Name: "first"},
	}

	if err := repository.Save(initial); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	const goroutines = 10

	done := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := repository.GetAll()
			done <- err
		}()
	}

	for i := 0; i < goroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent GetAll() error = %v", err)
		}
	}
}

func TestJSONRepository_GetByID_UsesIDGetter(t *testing.T) {
	repository, path := newTestRepository(t)

	items := []testItem{
		{ID: "device-1", Name: "Device One"},
		{ID: "device-2", Name: "Device Two"},
	}

	writeTestFile(t, path, items)

	got, err := repository.GetByID(
		"Device Two",
		func(item testItem) string {
			return item.Name
		},
	)

	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	want := testItem{
		ID:   "device-2",
		Name: "Device Two",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetByID() = %v, want %v", got, want)
	}
}

func TestJSONRepository_Update_PreservesOtherItems(t *testing.T) {
	repository, path := newTestRepository(t)

	initial := []testItem{
		{ID: "1", Name: "first"},
		{ID: "2", Name: "second"},
		{ID: "3", Name: "third"},
	}

	writeTestFile(t, path, initial)

	err := repository.Update(
		"2",
		testItem{
			ID:   "2",
			Name: "updated",
		},
		testItemID,
	)

	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repository.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	want := []testItem{
		{ID: "1", Name: "first"},
		{ID: "2", Name: "updated"},
		{ID: "3", Name: "third"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAll() = %v, want %v", got, want)
	}
}

func TestJSONRepository_Delete_PreservesOtherItems(t *testing.T) {
	repository, path := newTestRepository(t)

	initial := []testItem{
		{ID: "1", Name: "first"},
		{ID: "2", Name: "second"},
		{ID: "3", Name: "third"},
	}

	writeTestFile(t, path, initial)

	err := repository.Delete("2", testItemID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, err := repository.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	want := []testItem{
		{ID: "1", Name: "first"},
		{ID: "3", Name: "third"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAll() = %v, want %v", got, want)
	}
}

func TestJSONRepository_Save_ValidJSON(t *testing.T) {
	repository, path := newTestRepository(t)

	items := []testItem{
		{ID: "1", Name: "first"},
	}

	if err := repository.Save(items); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var decoded []testItem

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved file contains invalid JSON: %v", err)
	}

	if !reflect.DeepEqual(decoded, items) {
		t.Errorf("decoded = %v, want %v", decoded, items)
	}
}

func TestJSONRepository_Save_DoesNotLeaveTemporaryFiles(t *testing.T) {
	dataDir := t.TempDir()

	repository := NewJSONRepository[testItem](
		dataDir,
		"items.json",
	)

	items := []testItem{
		{ID: "1", Name: "first"},
	}

	if err := repository.Save(items); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf(
			"expected exactly one file, got %d",
			len(entries),
		)
	}

	if entries[0].Name() != "items.json" {
		t.Errorf(
			"unexpected file = %q",
			entries[0].Name(),
		)
	}
}

func TestJSONRepository_ErrorWrapping(t *testing.T) {
	repository, path := newTestRepository(t)

	if err := os.WriteFile(
		path,
		[]byte(`{invalid}`),
		0644,
	); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := repository.GetAll()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var syntaxError *json.SyntaxError
	if !errors.As(err, &syntaxError) {
		t.Fatalf("expected wrapped JSON syntax error, got %v", err)
	}
}
