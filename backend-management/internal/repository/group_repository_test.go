package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

func setupGroupTest(t *testing.T) (*repository.PostgresGroupRepository, func()) {
	t.Helper()
	ctx := context.Background()

	err := testutil.CleanupTables(ctx, testContainer.DB)
	require.NoError(t, err)

	repo := repository.NewPostgresGroupRepository(testContainer.DB)

	return repo, func() {}
}

func TestGroupRepository_Create(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	group := testutil.NewGroupBuilder().
		WithName("Test Group").
		WithDescription("A test group").
		WithColor("#FF5733").
		Build()

	err := repo.Create(ctx, group)
	require.NoError(t, err)

	assert.NotZero(t, group.ID)
	assert.NotZero(t, group.CreatedAt)
	assert.NotZero(t, group.UpdatedAt)
}

func TestGroupRepository_Create_DuplicateName(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	group1 := testutil.NewGroupBuilder().WithName("Duplicate").Build()
	err := repo.Create(ctx, group1)
	require.NoError(t, err)

	group2 := testutil.NewGroupBuilder().WithName("Duplicate").Build()
	err = repo.Create(ctx, group2)
	assert.Error(t, err, "should fail on duplicate name")
}

func TestGroupRepository_GetByID(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	group := testutil.NewGroupBuilder().
		WithName("GetByID Test").
		WithDescription("Test description").
		WithColor("#00FF00").
		Build()
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, group.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, group.ID, found.ID)
	assert.Equal(t, "GetByID Test", found.Name)
	assert.NotNil(t, found.Description)
	assert.Equal(t, "Test description", *found.Description)
	assert.Equal(t, "#00FF00", found.Color)
}

func TestGroupRepository_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	found, err := repo.GetByID(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestGroupRepository_List(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create multiple groups
	_, err := testutil.NewGroupBuilder().WithName("Alpha Group").Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewGroupBuilder().WithName("Beta Group").Create(ctx, testContainer.DB)
	require.NoError(t, err)

	_, err = testutil.NewGroupBuilder().WithName("Gamma Group").Create(ctx, testContainer.DB)
	require.NoError(t, err)

	groups, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, groups, 3)

	// Should be ordered by name
	assert.Equal(t, "Alpha Group", groups[0].Name)
	assert.Equal(t, "Beta Group", groups[1].Name)
	assert.Equal(t, "Gamma Group", groups[2].Name)
}

func TestGroupRepository_Update(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	group := testutil.NewGroupBuilder().
		WithName("Original Name").
		WithDescription("Original description").
		WithColor("#000000").
		Build()
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	// Update fields
	group.Name = "Updated Name"
	newDesc := "Updated description"
	group.Description = &newDesc
	group.Color = "#FFFFFF"

	updated, err := repo.Update(ctx, group)
	require.NoError(t, err)

	assert.Equal(t, "Updated Name", updated.Name)
	assert.NotNil(t, updated.Description)
	assert.Equal(t, "Updated description", *updated.Description)
	assert.Equal(t, "#FFFFFF", updated.Color)
	assert.True(t, updated.UpdatedAt.After(group.CreatedAt) || updated.UpdatedAt.Equal(group.CreatedAt))
}

func TestGroupRepository_Delete(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	group := testutil.NewGroupBuilder().WithName("To Delete").Build()
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(ctx, group.ID)
	require.NoError(t, err)

	// Verify deleted
	found, err := repo.GetByID(ctx, group.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}

func TestGroupRepository_Delete_WithAssignments(t *testing.T) {
	repo, cleanup := setupGroupTest(t)
	defer cleanup()
	ctx := context.Background()

	// Create group and employee
	group, err := testutil.NewGroupBuilder().WithName("Group with Members").Create(ctx, testContainer.DB)
	require.NoError(t, err)

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("member@example.com").
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Assign employee to group
	_, err = testutil.NewGroupAssignmentBuilder().
		WithEmployeeID(employee.ID).
		WithGroupID(group.ID).
		Create(ctx, testContainer.DB)
	require.NoError(t, err)

	// Delete group - should cascade to assignments
	err = repo.Delete(ctx, group.ID)
	require.NoError(t, err)

	// Verify group deleted
	found, err := repo.GetByID(ctx, group.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, found)
}
