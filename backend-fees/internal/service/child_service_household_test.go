package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestChildService_LinkParentRejectsDifferentHouseholds(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	parentRepo := repository.NewPostgresParentRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childService := service.NewChildService(childRepo, parentRepo, householdRepo)

	childHousehold := createHouseholdForTest(t, householdRepo, "TEST child household")
	parentHousehold := createHouseholdForTest(t, householdRepo, "TEST parent household")

	child := createChildForHouseholdTest(t, childRepo, "THM", childHousehold.ID)
	parent := createParentForHouseholdTest(t, parentRepo, "Parent", "Mismatch", parentHousehold.ID)

	err := childService.LinkParent(ctx, child.ID, parent.ID, false)
	if !errors.Is(err, service.ErrHouseholdMismatch) {
		t.Fatalf("expected ErrHouseholdMismatch, got %v", err)
	}

	parents, err := childRepo.GetParents(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetParents failed: %v", err)
	}
	if len(parents) != 0 {
		t.Fatalf("expected no linked parents, got %d", len(parents))
	}
}

func TestChildService_LinkParentAssignsChildToExistingParentHousehold(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	parentRepo := repository.NewPostgresParentRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)
	childService := service.NewChildService(childRepo, parentRepo, householdRepo)

	parentHousehold := createHouseholdForTest(t, householdRepo, "TEST parent household")
	child := createChildForHouseholdTest(t, childRepo, "THA", uuid.Nil)
	parent := createParentForHouseholdTest(t, parentRepo, "Parent", "Existing", parentHousehold.ID)

	if err := childService.LinkParent(ctx, child.ID, parent.ID, true); err != nil {
		t.Fatalf("LinkParent failed: %v", err)
	}

	loadedChild, err := childRepo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if loadedChild.HouseholdID == nil || *loadedChild.HouseholdID != parentHousehold.ID {
		t.Fatalf("expected child household %s, got %v", parentHousehold.ID, loadedChild.HouseholdID)
	}
}

func TestChildParentHouseholdTriggerRejectsDirectMismatch(t *testing.T) {
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	parentRepo := repository.NewPostgresParentRepository(testDB)
	householdRepo := repository.NewPostgresHouseholdRepository(testDB)

	childHousehold := createHouseholdForTest(t, householdRepo, "TEST trigger child household")
	parentHousehold := createHouseholdForTest(t, householdRepo, "TEST trigger parent household")

	child := createChildForHouseholdTest(t, childRepo, "THT", childHousehold.ID)
	parent := createParentForHouseholdTest(t, parentRepo, "Parent", "Trigger", parentHousehold.ID)

	err := childRepo.LinkParent(ctx, child.ID, parent.ID, false)
	if err == nil {
		t.Fatal("expected trigger to reject mismatched households")
	}
}

func createHouseholdForTest(t *testing.T, repo repository.HouseholdRepository, name string) *domain.Household {
	t.Helper()

	household := &domain.Household{
		ID:               uuid.New(),
		Name:             name,
		MembershipStatus: domain.MembershipAssignmentStatusAssumed,
	}
	if err := repo.Create(context.Background(), household); err != nil {
		t.Fatalf("household Create failed: %v", err)
	}
	return household
}

func createChildForHouseholdTest(t *testing.T, repo repository.ChildRepository, suffix string, householdID uuid.UUID) *domain.Child {
	t.Helper()

	child, err := createTestChild(repo, suffix)
	if err != nil {
		t.Fatal(err)
	}
	if householdID != uuid.Nil {
		child.HouseholdID = &householdID
		if err := repo.Update(context.Background(), child); err != nil {
			t.Fatalf("child Update failed: %v", err)
		}
	}
	return child
}

func createParentForHouseholdTest(t *testing.T, repo repository.ParentRepository, firstName string, lastName string, householdID uuid.UUID) *domain.Parent {
	t.Helper()

	parent := &domain.Parent{
		ID:          uuid.New(),
		HouseholdID: &householdID,
		FirstName:   firstName,
		LastName:    lastName,
		Email:       stringPtr(firstName + "." + lastName + "@example.test"),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Create(context.Background(), parent); err != nil {
		t.Fatalf("parent Create failed: %v", err)
	}
	return parent
}
