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

func setupScheduleTest(t *testing.T) (*repository.PostgresScheduleRepository, *domain.Employee, *domain.Group, func()) {
	t.Helper()
	ctx := context.Background()

	err := testutil.CleanupTables(ctx, testContainer.DB)
	require.NoError(t, err)

	// Create an employee and group for schedule entries
	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("schedule@example.com").
		WithName("Schedule", "Tester").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	repo := repository.NewPostgresScheduleRepository(testContainer.DB)

	return repo, employee, group, func() {}
}

func TestScheduleRepository_Create(t *testing.T) {
	repo, employee, group, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	startTime := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
	endTime := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

	entry := &domain.ScheduleEntry{
		EmployeeID:   employee.ID,
		Date:         today,
		StartTime:    &startTime,
		EndTime:      &endTime,
		BreakMinutes: 30,
		GroupID:      &group.ID,
		EntryType:    domain.ScheduleEntryTypeWork,
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)

	assert.NotZero(t, entry.ID)
	assert.NotZero(t, entry.CreatedAt)
	assert.NotZero(t, entry.UpdatedAt)
}

func TestScheduleRepository_Create_WithNotes(t *testing.T) {
	repo, employee, _, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	notes := "Morning meeting"

	entry := &domain.ScheduleEntry{
		EmployeeID: employee.ID,
		Date:       today,
		EntryType:  domain.ScheduleEntryTypeEvent,
		Notes:      &notes,
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)
	assert.NotZero(t, entry.ID)
}

func TestScheduleRepository_GetByID(t *testing.T) {
	repo, employee, group, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	startTime := time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(today.Year(), today.Month(), today.Day(), 17, 0, 0, 0, time.UTC)
	notes := "Test notes"

	entry, err := testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithTimes(startTime, endTime).
		WithBreak(45).
		WithGroupID(group.ID).
		WithType(domain.ScheduleEntryTypeWork).
		WithNotes(notes).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, entry.ID, found.ID)
	assert.Equal(t, employee.ID, found.EmployeeID)
	assert.Equal(t, 45, found.BreakMinutes)
	assert.Equal(t, domain.ScheduleEntryTypeWork, found.EntryType)
	assert.NotNil(t, found.Notes)
	assert.Equal(t, "Test notes", *found.Notes)

	// Check relations are loaded
	assert.NotNil(t, found.Employee)
	assert.Equal(t, employee.ID, found.Employee.ID)
	assert.Equal(t, "Schedule", found.Employee.FirstName)

	assert.NotNil(t, found.Group)
	assert.Equal(t, group.ID, found.Group.ID)
	assert.Equal(t, "Test Group", found.Group.Name)
}

func TestScheduleRepository_GetByID_NotFound(t *testing.T) {
	repo, _, _, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	found, err := repo.GetByID(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestScheduleRepository_List(t *testing.T) {
	repo, employee, group, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)

	// Create entries for multiple days
	_, err := testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(tomorrow).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(dayAfter).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// List all entries
	entries, err := repo.List(ctx, today, dayAfter, nil, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// List with date filter
	entries, err = repo.List(ctx, today, tomorrow, nil, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestScheduleRepository_List_FilterByEmployee(t *testing.T) {
	repo, employee1, group, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create second employee
	employee2, err := testutil.NewEmployeeBuilder().
		WithEmail("employee2@example.com").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)

	// Create entries for both employees
	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee1.ID).
		WithDate(today).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee1.ID).
		WithDate(today.AddDate(0, 0, 1)).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee2.ID).
		WithDate(today).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Filter by employee1
	entries, err := repo.List(ctx, today, today.AddDate(0, 0, 7), &employee1.ID, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Filter by employee2
	entries, err = repo.List(ctx, today, today.AddDate(0, 0, 7), &employee2.ID, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestScheduleRepository_List_FilterByGroup(t *testing.T) {
	repo, employee, group1, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create second group
	group2, err := testutil.NewGroupBuilder().
		WithName("Second Group").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	today := time.Now().Truncate(24 * time.Hour)

	// Create entries in different groups
	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithGroupID(group1.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today.AddDate(0, 0, 1)).
		WithGroupID(group2.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Filter by group1
	entries, err := repo.List(ctx, today, today.AddDate(0, 0, 7), nil, &group1.ID)
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// Filter by group2
	entries, err = repo.List(ctx, today, today.AddDate(0, 0, 7), nil, &group2.ID)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestScheduleRepository_Update(t *testing.T) {
	repo, employee, group, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	startTime := time.Date(today.Year(), today.Month(), today.Day(), 8, 0, 0, 0, time.UTC)
	endTime := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC)

	entry, err := testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithDate(today).
		WithTimes(startTime, endTime).
		WithBreak(30).
		WithGroupID(group.ID).
		WithType(domain.ScheduleEntryTypeWork).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Update entry
	newStartTime := time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, time.UTC)
	newEndTime := time.Date(today.Year(), today.Month(), today.Day(), 18, 0, 0, 0, time.UTC)
	notes := "Updated schedule"

	entry.StartTime = &newStartTime
	entry.EndTime = &newEndTime
	entry.BreakMinutes = 60
	entry.EntryType = domain.ScheduleEntryTypeTraining
	entry.Notes = &notes

	updated, err := repo.Update(ctx, entry)
	require.NoError(t, err)

	assert.Equal(t, 9, updated.StartTime.Hour())
	assert.Equal(t, 18, updated.EndTime.Hour())
	assert.Equal(t, 60, updated.BreakMinutes)
	assert.Equal(t, domain.ScheduleEntryTypeTraining, updated.EntryType)
	assert.NotNil(t, updated.Notes)
	assert.Equal(t, "Updated schedule", *updated.Notes)
}

func TestScheduleRepository_Delete(t *testing.T) {
	repo, employee, group, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	entry, err := testutil.NewScheduleEntryBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	err = repo.Delete(ctx, entry.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, entry.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestScheduleRepository_DifferentEntryTypes(t *testing.T) {
	repo, employee, _, cleanup := setupScheduleTest(t)
	defer cleanup()
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	entryTypes := []domain.ScheduleEntryType{
		domain.ScheduleEntryTypeWork,
		domain.ScheduleEntryTypeVacation,
		domain.ScheduleEntryTypeSick,
		domain.ScheduleEntryTypeSpecialLeave,
		domain.ScheduleEntryTypeTraining,
		domain.ScheduleEntryTypeEvent,
	}

	for i, entryType := range entryTypes {
		entry, err := testutil.NewScheduleEntryBuilder().
			WithEmployeeID(employee.ID).
			WithDate(today.AddDate(0, 0, i)).
			WithType(entryType).
			Create(ctx, testContainer.DB)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entryType, found.EntryType, "entry type mismatch for %s", entryType)
	}
}
