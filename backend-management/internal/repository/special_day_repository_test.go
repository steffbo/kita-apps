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

func setupSpecialDayTest(t *testing.T) (*repository.PostgresSpecialDayRepository, func()) {
	t.Helper()
	ctx := context.Background()

	err := testutil.CleanupTables(ctx, testContainer.DB)
	require.NoError(t, err)

	repo := repository.NewPostgresSpecialDayRepository(testContainer.DB)

	return repo, func() {}
}

func TestSpecialDayRepository_Create(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	date := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
	day := &domain.SpecialDay{
		Date:       date,
		Name:       "Christmas",
		DayType:    domain.SpecialDayTypeHoliday,
		AffectsAll: true,
	}

	err := repo.Create(ctx, day)
	require.NoError(t, err)

	assert.NotZero(t, day.ID)
	assert.NotZero(t, day.CreatedAt)
}

func TestSpecialDayRepository_Create_WithEndDate(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	startDate := time.Date(2024, 12, 23, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	notes := "Winter break closure"

	day := &domain.SpecialDay{
		Date:       startDate,
		EndDate:    &endDate,
		Name:       "Winter Closure",
		DayType:    domain.SpecialDayTypeClosure,
		AffectsAll: true,
		Notes:      &notes,
	}

	err := repo.Create(ctx, day)
	require.NoError(t, err)
	assert.NotZero(t, day.ID)
}

func TestSpecialDayRepository_GetByID(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	date := time.Date(2024, 10, 3, 0, 0, 0, 0, time.UTC)
	notes := "German Unity Day"

	day, err := testutil.NewSpecialDayBuilder().
		WithDate(date).
		WithName("Tag der Deutschen Einheit").
		WithType(domain.SpecialDayTypeHoliday).
		AffectsAll(true).
		WithNotes(notes).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, day.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, day.ID, found.ID)
	assert.Equal(t, "Tag der Deutschen Einheit", found.Name)
	assert.Equal(t, domain.SpecialDayTypeHoliday, found.DayType)
	assert.True(t, found.AffectsAll)
	assert.NotNil(t, found.Notes)
	assert.Equal(t, "German Unity Day", *found.Notes)
}

func TestSpecialDayRepository_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	found, err := repo.GetByID(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestSpecialDayRepository_List(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create multiple special days
	dates := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
	}
	names := []string{"New Year", "May Day", "Christmas"}

	for i, date := range dates {
		_, err := testutil.NewSpecialDayBuilder().
			WithDate(date).
			WithName(names[i]).
			WithType(domain.SpecialDayTypeHoliday).
			Create(ctx, testContainer.DB)
		require.NoError(t, err)
	}

	// List entire year
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	days, err := repo.List(ctx, startDate, endDate)
	require.NoError(t, err)
	assert.Len(t, days, 3)

	// Should be ordered by date
	assert.Equal(t, "New Year", days[0].Name)
	assert.Equal(t, "May Day", days[1].Name)
	assert.Equal(t, "Christmas", days[2].Name)

	// List partial range
	days, err = repo.List(ctx, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Len(t, days, 1)
	assert.Equal(t, "May Day", days[0].Name)
}

func TestSpecialDayRepository_ListByType(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	year := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endYear := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	// Create different types of special days
	_, err := testutil.NewSpecialDayBuilder().
		WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
		WithName("New Year").
		WithType(domain.SpecialDayTypeHoliday).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewSpecialDayBuilder().
		WithDate(time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)).
		WithName("Summer Closure").
		WithType(domain.SpecialDayTypeClosure).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewSpecialDayBuilder().
		WithDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)).
		WithName("Team Building").
		WithType(domain.SpecialDayTypeTeamDay).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewSpecialDayBuilder().
		WithDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)).
		WithName("Kids Festival").
		WithType(domain.SpecialDayTypeEvent).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// List by type
	holidays, err := repo.ListByType(ctx, year, endYear, domain.SpecialDayTypeHoliday)
	require.NoError(t, err)
	assert.Len(t, holidays, 1)
	assert.Equal(t, "New Year", holidays[0].Name)

	closures, err := repo.ListByType(ctx, year, endYear, domain.SpecialDayTypeClosure)
	require.NoError(t, err)
	assert.Len(t, closures, 1)
	assert.Equal(t, "Summer Closure", closures[0].Name)

	teamDays, err := repo.ListByType(ctx, year, endYear, domain.SpecialDayTypeTeamDay)
	require.NoError(t, err)
	assert.Len(t, teamDays, 1)
	assert.Equal(t, "Team Building", teamDays[0].Name)

	events, err := repo.ListByType(ctx, year, endYear, domain.SpecialDayTypeEvent)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "Kids Festival", events[0].Name)
}

func TestSpecialDayRepository_Update(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	day, err := testutil.NewSpecialDayBuilder().
		WithDate(time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)).
		WithName("Christmas").
		WithType(domain.SpecialDayTypeHoliday).
		AffectsAll(true).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Update fields
	day.Name = "Weihnachten"
	endDate := time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)
	day.EndDate = &endDate
	notes := "Two-day holiday"
	day.Notes = &notes

	updated, err := repo.Update(ctx, day)
	require.NoError(t, err)

	assert.Equal(t, "Weihnachten", updated.Name)
	assert.NotNil(t, updated.EndDate)
	assert.NotNil(t, updated.Notes)
	assert.Equal(t, "Two-day holiday", *updated.Notes)
}

func TestSpecialDayRepository_Delete(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	day, err := testutil.NewSpecialDayBuilder().
		WithDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
		WithName("To Delete").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	err = repo.Delete(ctx, day.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, day.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestSpecialDayRepository_AllDayTypes(t *testing.T) {
	repo, cleanup := setupSpecialDayTest(t)
	defer cleanup()
	ctx := context.Background()

	dayTypes := []domain.SpecialDayType{
		domain.SpecialDayTypeHoliday,
		domain.SpecialDayTypeClosure,
		domain.SpecialDayTypeTeamDay,
		domain.SpecialDayTypeEvent,
	}

	for i, dayType := range dayTypes {
		day, err := testutil.NewSpecialDayBuilder().
			WithDate(time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC)).
			WithName(string(dayType)).
			WithType(dayType).
			Create(ctx, testContainer.DB)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, day.ID)
		require.NoError(t, err)
		assert.Equal(t, dayType, found.DayType, "day type mismatch for %s", dayType)
	}
}
