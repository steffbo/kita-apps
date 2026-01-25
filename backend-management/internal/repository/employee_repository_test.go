package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

var testContainer *testutil.TestContainer

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testContainer, err = testutil.SetupPostgres(ctx)
	if err != nil {
		panic("failed to setup postgres: " + err.Error())
	}

	code := m.Run()

	testContainer.Cleanup(ctx)
	if code != 0 {
		panic("tests failed")
	}
}

func setupEmployeeTest(t *testing.T) (*repository.PostgresEmployeeRepository, func()) {
	t.Helper()
	ctx := context.Background()

	// Clean up before test
	err := testutil.CleanupTables(ctx, testContainer.DB)
	require.NoError(t, err)

	repo := repository.NewPostgresEmployeeRepository(testContainer.DB)

	return repo, func() {
		// Cleanup is handled by the next test's setup
	}
}

func TestEmployeeRepository_Create(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("create@example.com").
		WithName("Create", "Test").
		Build()

	err := repo.Create(ctx, employee)
	require.NoError(t, err)

	assert.NotZero(t, employee.ID, "employee ID should be set after create")
	assert.NotZero(t, employee.CreatedAt, "created_at should be set")
	assert.NotZero(t, employee.UpdatedAt, "updated_at should be set")
}

func TestEmployeeRepository_Create_DuplicateEmail(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create first employee
	employee1 := testutil.NewEmployeeBuilder().
		WithEmail("duplicate@example.com").
		Build()
	err := repo.Create(ctx, employee1)
	require.NoError(t, err)

	// Try to create second with same email
	employee2 := testutil.NewEmployeeBuilder().
		WithEmail("duplicate@example.com").
		Build()
	err = repo.Create(ctx, employee2)
	assert.Error(t, err, "should fail on duplicate email")
}

func TestEmployeeRepository_GetByID(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create employee
	employee := testutil.NewEmployeeBuilder().
		WithEmail("getbyid@example.com").
		WithName("GetByID", "Test").
		WithWeeklyHours(35.0).
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)

	// Retrieve by ID
	found, err := repo.GetByID(ctx, employee.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, employee.ID, found.ID)
	assert.Equal(t, "getbyid@example.com", found.Email)
	assert.Equal(t, "GetByID", found.FirstName)
	assert.Equal(t, "Test", found.LastName)
	assert.Equal(t, 35.0, found.WeeklyHours)
}

func TestEmployeeRepository_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	found, err := repo.GetByID(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestEmployeeRepository_GetByEmail(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("getbyemail@example.com").
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)

	found, err := repo.GetByEmail(ctx, "getbyemail@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, employee.ID, found.ID)
}

func TestEmployeeRepository_GetByEmail_NotFound(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	found, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestEmployeeRepository_ExistsByEmail(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("exists@example.com").
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)

	exists, err := repo.ExistsByEmail(ctx, "exists@example.com")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByEmail(ctx, "notexists@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestEmployeeRepository_List(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create multiple employees
	_, err := testutil.NewEmployeeBuilder().
		WithEmail("active1@example.com").
		WithName("Active", "One").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("active2@example.com").
		WithName("Active", "Two").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewEmployeeBuilder().
		WithEmail("inactive@example.com").
		WithName("Inactive", "User").
		Inactive().
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// List all
	all, err := repo.List(ctx, false)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// List active only
	active, err := repo.List(ctx, true)
	require.NoError(t, err)
	assert.Len(t, active, 2)
}

func TestEmployeeRepository_Update(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("update@example.com").
		WithName("Original", "Name").
		WithWeeklyHours(40.0).
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)

	// Update fields
	employee.FirstName = "Updated"
	employee.LastName = "User"
	employee.WeeklyHours = 30.0
	employee.Role = domain.EmployeeRoleAdmin

	updated, err := repo.Update(ctx, employee)
	require.NoError(t, err)

	assert.Equal(t, "Updated", updated.FirstName)
	assert.Equal(t, "User", updated.LastName)
	assert.Equal(t, 30.0, updated.WeeklyHours)
	assert.Equal(t, domain.EmployeeRoleAdmin, updated.Role)
	assert.True(t, updated.UpdatedAt.After(employee.CreatedAt))
}

func TestEmployeeRepository_UpdatePassword(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("password@example.com").
		WithPassword("OldPassword123!").
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)

	originalHash := employee.PasswordHash

	// Update password
	newHash := "new-password-hash"
	err = repo.UpdatePassword(ctx, employee.ID, newHash)
	require.NoError(t, err)

	// Verify password changed
	found, err := repo.GetByID(ctx, employee.ID)
	require.NoError(t, err)
	assert.NotEqual(t, originalHash, found.PasswordHash)
	assert.Equal(t, newHash, found.PasswordHash)
}

func TestEmployeeRepository_Deactivate(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("deactivate@example.com").
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)
	assert.True(t, employee.Active)

	// Deactivate
	err = repo.Deactivate(ctx, employee.ID)
	require.NoError(t, err)

	// Verify deactivated
	found, err := repo.GetByID(ctx, employee.ID)
	require.NoError(t, err)
	assert.False(t, found.Active)
}

func TestEmployeeRepository_AdjustRemainingVacationDays(t *testing.T) {
	repo, cleanup := setupEmployeeTest(t)
	defer cleanup()
	ctx := context.Background()

	employee := testutil.NewEmployeeBuilder().
		WithEmail("vacation@example.com").
		WithVacationDays(30, 25.0).
		Build()
	err := repo.Create(ctx, employee)
	require.NoError(t, err)
	assert.Equal(t, 25.0, employee.RemainingVacationDays)

	// Subtract vacation days (taking vacation)
	err = repo.AdjustRemainingVacationDays(ctx, employee.ID, -5.0)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, employee.ID)
	require.NoError(t, err)
	assert.Equal(t, 20.0, found.RemainingVacationDays)

	// Add vacation days
	err = repo.AdjustRemainingVacationDays(ctx, employee.ID, 2.0)
	require.NoError(t, err)

	found, err = repo.GetByID(ctx, employee.ID)
	require.NoError(t, err)
	assert.Equal(t, 22.0, found.RemainingVacationDays)
}
