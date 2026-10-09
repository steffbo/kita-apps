package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

func TestChildImport_RollsBackFailedRowAndContinues(t *testing.T) {
	requireTestDatabase(t)
	cleanupTestData()
	defer cleanupTestData()

	ctx := context.Background()
	childRepo := repository.NewPostgresChildRepository(testDB)
	parentRepo := repository.NewPostgresParentRepository(testDB)
	svc := service.NewChildImportService(childRepo, parentRepo, repository.NewTxManager(testDB))

	newRow := func(index int, memberNumber string) service.ImportRow {
		return service.ImportRow{
			Index: index,
			Child: service.ChildPreview{
				MemberNumber: memberNumber,
				FirstName:    "Import",
				LastName:     "Test",
				BirthDate:    "2022-01-01",
				EntryDate:    "2024-01-01",
				CareHours:    intPtr(6),
				LegalHours:   intPtr(6),
			},
		}
	}
	failed := newRow(1, "TIMPFAIL")
	failed.Parent1 = &service.ParentPreview{
		FirstName: "Rollback", LastName: "Parent", Email: "import-rollback@example.test",
	}
	failed.Parent2 = &service.ParentPreview{FirstName: "Missing", LastName: "Parent"}
	success := newRow(2, "TIMPAFTER")
	success.Parent1 = &service.ParentPreview{
		FirstName: "Committed", LastName: "Parent", Email: "import-committed@example.test",
	}

	result, err := svc.Execute(ctx, &service.ExecuteRequest{
		Rows: []service.ImportRow{newRow(0, "TIMPBEFORE"), failed, success},
		ParentDecisions: []service.ParentDecision{{
			RowIndex: 1, ParentIndex: 2, Action: "link", ExistingParentID: uuid.NewString(),
		}},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(result.Errors) != 1 || result.Errors[0].RowIndex != 1 ||
		!strings.HasPrefix(result.Errors[0].Error, "Fehler beim Verknüpfen von Elternteil 2: ") {
		t.Fatalf("unexpected import errors: %+v", result.Errors)
	}
	// Counters retain their existing meaning, including completed steps of failed rows.
	if result.ChildrenCreated != 3 || result.ChildrenUpdated != 0 ||
		result.ParentsCreated != 2 || result.ParentsLinked != 0 {
		t.Fatalf("unexpected counters: %+v", result)
	}

	for memberNumber, want := range map[string]int{"TIMPBEFORE": 1, "TIMPFAIL": 0, "TIMPAFTER": 1} {
		var count int
		if err := testDB.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM fees.children WHERE member_number = $1`, memberNumber); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("children for %s = %d, want %d", memberNumber, count, want)
		}
	}
	for email, want := range map[string]int{
		"import-rollback@example.test": 0, "import-committed@example.test": 1,
	} {
		var count int
		if err := testDB.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM fees.parents WHERE email = $1`, email); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("parents for %s = %d, want %d", email, count, want)
		}
	}
	child, err := childRepo.GetByMemberNumber(ctx, "TIMPAFTER")
	if err != nil {
		t.Fatal(err)
	}
	parents, err := childRepo.GetParents(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(parents) != 1 || parents[0].Email == nil || *parents[0].Email != success.Parent1.Email {
		t.Fatalf("unexpected committed parent links: %+v", parents)
	}
}
