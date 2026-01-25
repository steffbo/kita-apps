package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

func setupTimeEntryTest(t *testing.T) (*repository.PostgresTimeEntryRepository, *domain.Employee, func()) {
	t.Helper()
	ctx := context.Background()

	err := testutil.CleanupTables(ctx, testContainer.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("timeentry@example.com").
		WithName("Time", "Tracker").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	repo := repository.NewPostgresTimeEntryRepository(testContainer.DB)

	return repo, employee, func() {}
}

func TestTimeEntryRepository_Create(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)

	entry := &domain.TimeEntry{
		EmployeeID:   employee.ID,
		Date:         today,
		ClockIn:      clockIn,
		BreakMinutes: 0,
		EntryType:    domain.TimeEntryTypeWork,
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)

	assert.NotZero(t, entry.ID)
	assert.NotZero(t, entry.CreatedAt)
}

func TestTimeEntryRepository_Create_WithClockOut(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)
	clockOut := time.Date(now.Year(), now.Month(), now.Day(), 16, 30, 0, 0, time.UTC)

	entry := &domain.TimeEntry{
		EmployeeID:   employee.ID,
		Date:         today,
		ClockIn:      clockIn,
		ClockOut:     &clockOut,
		BreakMinutes: 30,
		EntryType:    domain.TimeEntryTypeWork,
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)
	assert.NotZero(t, entry.ID)
}

func TestTimeEntryRepository_GetByID(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC)
	clockOut := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, time.UTC)
	notes := "Regular workday"

	entry, err := testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		WithClockOut(clockOut).
		WithBreak(45).
		WithType(domain.TimeEntryTypeWork).
		WithNotes(notes).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, entry.ID, found.ID)
	assert.Equal(t, employee.ID, found.EmployeeID)
	assert.Equal(t, 45, found.BreakMinutes)
	assert.Equal(t, domain.TimeEntryTypeWork, found.EntryType)
	assert.NotNil(t, found.Notes)
	assert.Equal(t, "Regular workday", *found.Notes)
	assert.NotNil(t, found.ClockOut)

	// Check employee relation
	assert.NotNil(t, found.Employee)
	assert.Equal(t, employee.ID, found.Employee.ID)
	assert.Equal(t, "Time", found.Employee.FirstName)
}

func TestTimeEntryRepository_GetByID_NotFound(t *testing.T) {
	repo, _, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	found, err := repo.GetByID(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestTimeEntryRepository_List(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)

	// Create entries for multiple days
	for _, date := range []time.Time{today, tomorrow, dayAfter} {
		clockIn := time.Date(date.Year(), date.Month(), date.Day(), 8, 0, 0, 0, time.UTC)
		_, err := testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(date).
			WithClockIn(clockIn).
			Create(ctx, testContainer.DB)
		require.NoError(t, err)
	}

	// List all entries
	entries, err := repo.List(ctx, today, dayAfter, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// List with date range
	entries, err = repo.List(ctx, today, tomorrow, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestTimeEntryRepository_List_FilterByEmployee(t *testing.T) {
	repo, employee1, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create second employee
	employee2, err := testutil.NewEmployeeBuilder().
		WithEmail("employee2@example.com").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)

	// Create entries for both employees
	_, err = testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee1.ID).
		WithDate(today).
		WithClockIn(clockIn).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee1.ID).
		WithDate(today.AddDate(0, 0, 1)).
		WithClockIn(clockIn.AddDate(0, 0, 1)).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee2.ID).
		WithDate(today).
		WithClockIn(clockIn).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Filter by employee1
	entries, err := repo.List(ctx, today, today.AddDate(0, 0, 7), &employee1.ID)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Filter by employee2
	entries, err = repo.List(ctx, today, today.AddDate(0, 0, 7), &employee2.ID)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestTimeEntryRepository_ListOpenByEmployeeID(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)
	clockOut := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, time.UTC)

	// Create open entry (no clock_out)
	_, err := testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Create closed entry (with clock_out)
	_, err = testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today.AddDate(0, 0, -1)).
		WithClockIn(clockIn.AddDate(0, 0, -1)).
		WithClockOut(clockOut.AddDate(0, 0, -1)).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Should only return open entries
	openEntries, err := repo.ListOpenByEmployeeID(ctx, employee.ID)
	require.NoError(t, err)
	assert.Len(t, openEntries, 1)
	assert.Nil(t, openEntries[0].ClockOut)
}

func TestTimeEntryRepository_Update(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)

	entry, err := testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Update: clock out
	clockOut := time.Date(now.Year(), now.Month(), now.Day(), 16, 30, 0, 0, time.UTC)
	entry.ClockOut = &clockOut
	entry.BreakMinutes = 30
	notes := "Updated entry"
	entry.Notes = &notes

	updated, err := repo.Update(ctx, entry)
	require.NoError(t, err)

	assert.NotNil(t, updated.ClockOut)
	assert.Equal(t, 30, updated.BreakMinutes)
	assert.NotNil(t, updated.Notes)
	assert.Equal(t, "Updated entry", *updated.Notes)
}

func TestTimeEntryRepository_Update_WithEditInfo(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create admin for editing
	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)

	entry, err := testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Update with edit tracking
	clockOut := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, time.UTC)
	editedAt := time.Now()
	editReason := "Corrected clock-out time"

	entry.ClockOut = &clockOut
	entry.EditedBy = &admin.ID
	entry.EditedAt = &editedAt
	entry.EditReason = &editReason

	updated, err := repo.Update(ctx, entry)
	require.NoError(t, err)

	assert.NotNil(t, updated.EditedBy)
	assert.Equal(t, admin.ID, *updated.EditedBy)
	assert.NotNil(t, updated.EditedAt)
	assert.NotNil(t, updated.EditReason)
	assert.Equal(t, "Corrected clock-out time", *updated.EditReason)
}

func TestTimeEntryRepository_Delete(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	clockIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)

	entry, err := testutil.NewTimeEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithClockIn(clockIn).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	err = repo.Delete(ctx, entry.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, entry.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestTimeEntryRepository_DifferentEntryTypes(t *testing.T) {
	repo, employee, cleanup := setupTimeEntryTest(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	entryTypes := []domain.TimeEntryType{
		domain.TimeEntryTypeWork,
		domain.TimeEntryTypeVacation,
		domain.TimeEntryTypeSick,
		domain.TimeEntryTypeSpecialLeave,
		domain.TimeEntryTypeTraining,
		domain.TimeEntryTypeEvent,
	}

	for i, entryType := range entryTypes {
		date := today.AddDate(0, 0, i)
		clockIn := time.Date(date.Year(), date.Month(), date.Day(), 8, 0, 0, 0, time.UTC)

		entry, err := testutil.NewTimeEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(date).
			WithClockIn(clockIn).
			WithType(entryType).
			Create(ctx, testContainer.DB)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entryType, found.EntryType, "entry type mismatch for %s", entryType)
	}
}
