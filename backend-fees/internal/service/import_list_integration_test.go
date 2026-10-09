package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

func TestImportTransactionLists_SearchSortPagination(t *testing.T) {
	requireTestDatabase(t)
	cleanupTestData()
	t.Cleanup(cleanupTestData)
	ctx := context.Background()
	txRepo := repository.NewPostgresTransactionRepository(testDB)
	warningRepo := repository.NewPostgresWarningRepository(testDB)
	txs := make([]*domain.BankTransaction, 105)
	date := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for i := range txs {
		tx := &domain.BankTransaction{
			ID: uuid.New(), BookingDate: date.AddDate(0, 0, i), ValueDate: date,
			PayerName:   stringPtr(fmt.Sprintf("Payer %03d", i)),
			PayerIBAN:   stringPtr(fmt.Sprintf("LISTIBAN%03d", i)),
			Description: stringPtr(fmt.Sprintf("Invoice %03d", i)),
			Amount:      float64(i + 1), Currency: "EUR", ImportedAt: date,
		}
		if err := txRepo.Create(ctx, tx); err != nil {
			t.Fatal(err)
		}
		txs[i] = tx
	}
	childRepo := repository.NewPostgresChildRepository(testDB)
	child, err := createTestChild(childRepo, "LS")
	if err != nil {
		t.Fatal(err)
	}
	fee, err := createTestFee(repository.NewPostgresFeeRepository(testDB), child.ID,
		domain.FeeTypeFood, 0.5, 2026, 9)
	if err != nil {
		t.Fatal(err)
	}
	match := &domain.PaymentMatch{
		ID: uuid.New(), TransactionID: txs[0].ID, ExpectationID: fee.ID,
		Amount: 0.5, MatchType: domain.MatchTypeManual, MatchedAt: date,
	}
	if err := repository.NewPostgresMatchRepository(testDB).Create(ctx, match); err != nil {
		t.Fatal(err)
	}
	match.ID = uuid.New()
	match.TransactionID = txs[1].ID
	if err := repository.NewPostgresMatchRepository(testDB).Create(ctx, match); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 104} {
		warning := &domain.TransactionWarning{
			ID: uuid.New(), TransactionID: txs[index].ID,
			WarningType: domain.WarningTypeUnexpectedAmount, Message: "Review", CreatedAt: date,
		}
		if err := warningRepo.Create(ctx, warning); err != nil {
			t.Fatal(err)
		}
	}
	ignored := &domain.TransactionWarning{
		ID: uuid.New(), TransactionID: txs[2].ID,
		WarningType: domain.WarningTypeMultipleOpenFees, Message: "Ignored", CreatedAt: date,
	}
	if err := warningRepo.Create(ctx, ignored); err != nil {
		t.Fatal(err)
	}
	type listFunc func(string, string, string, int, int) ([]domain.BankTransaction, int64, error)
	lists := map[string]listFunc{
		"legacy unmatched": func(search, field, dir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
			return txRepo.ListUnmatched(ctx, search, field, dir, offset, limit)
		},
		"unified all": func(search, field, dir string, offset, limit int) ([]domain.BankTransaction, int64, error) {
			return txRepo.ListTransactions(ctx, "alle", search, field, dir, offset, limit)
		},
	}
	for name, list := range lists {
		t.Run(name, func(t *testing.T) {
			for _, search := range []string{"payer 104", "INVOICE 104", "listiban104"} {
				rows, total, err := list(search, "date", "desc", 0, 50)
				if err != nil {
					t.Fatal(err)
				}
				if total != 1 || len(rows) != 1 || rows[0].ID != txs[104].ID {
					t.Fatalf("search %q: total=%d rows=%v", search, total, rows)
				}
			}
			for _, field := range []string{"date", "payer", "description", "amount"} {
				for _, dir := range []string{"asc", "desc"} {
					first, total, err := list("", field, dir, 0, 100)
					if err != nil {
						t.Fatal(err)
					}
					last, lastTotal, err := list("", field, dir, 100, 100)
					if err != nil {
						t.Fatal(err)
					}
					if total != 105 || lastTotal != total || len(first) != 100 || len(last) != 5 {
						t.Fatalf("%s/%s: total=%d first=%d last=%d", field, dir, total, len(first), len(last))
					}
					rows := append(first, last...)
					for i, row := range rows {
						index := i
						if dir == "desc" {
							index = 104 - i
						}
						if row.ID != txs[index].ID {
							t.Fatalf("%s/%s row %d has wrong order", field, dir, i)
						}
					}
				}
			}
			rows, total, err := list("missing", "date", "desc", 0, 50)
			if err != nil || total != 0 || len(rows) != 0 {
				t.Fatalf("empty search: %v %d %v", err, total, rows)
			}
		})
	}
	for status, want := range map[string]int64{"offen": 103, "zugeordnet": 2, "warnungen": 2, "alle": 105} {
		rows, total, err := txRepo.ListTransactions(ctx, status, "", "amount", "asc", 0, 200)
		if err != nil {
			t.Fatal(err)
		}
		if total != want || int64(len(rows)) != want {
			t.Fatalf("%s total=%d want=%d", status, total, want)
		}
	}
	for _, field := range []string{"date", "payer", "description", "amount"} {
		for _, dir := range []string{"asc", "desc"} {
			rows, total, err := txRepo.ListMatched(ctx, "", field, dir, 1, 1)
			index := 1
			if dir == "desc" {
				index = 0
			}
			if err != nil || total != 2 || len(rows) != 1 || rows[0].ID != txs[index].ID {
				t.Fatalf("matched %s/%s: err=%v total=%d rows=%v", field, dir, err, total, rows)
			}
		}
	}
	matched, total, err := txRepo.ListMatched(ctx, "listiban000", "amount", "asc", 0, 1)
	if err != nil || total != 1 || len(matched) != 1 || matched[0].ID != txs[0].ID {
		t.Fatalf("matched IBAN search: err=%v total=%d rows=%v", err, total, matched)
	}
	matched, total, err = txRepo.ListMatched(ctx, "", "amount", "desc", 2, 1)
	if err != nil || total != 2 || len(matched) != 0 {
		t.Fatalf("matched empty page: err=%v total=%d rows=%v", err, total, matched)
	}
	for _, dir := range []string{"asc", "desc"} {
		option := repository.WarningListOptions{SortBy: "amount", SortDir: dir}
		rows, total, err := warningRepo.ListUnresolved(ctx, 1, 1, option)
		if err != nil {
			t.Fatal(err)
		}
		index := 104
		if dir == "desc" {
			index = 0
		}
		if total != 2 || len(rows) != 1 || rows[0].TransactionID != txs[index].ID {
			t.Fatalf("warnings %s: total=%d rows=%v", dir, total, rows)
		}
	}
	option := repository.WarningListOptions{Search: "listiban104"}
	warnings, total, err := warningRepo.ListUnresolved(ctx, 0, 100, option)
	if err != nil || total != 1 || len(warnings) != 1 || warnings[0].TransactionID != txs[104].ID {
		t.Fatalf("warning search: err=%v total=%d rows=%v", err, total, warnings)
	}
	option = repository.WarningListOptions{TransactionIDs: []uuid.UUID{txs[0].ID}}
	warnings, total, err = warningRepo.ListUnresolved(ctx, 0, 100, option)
	if err != nil || total != 1 || len(warnings) != 1 || warnings[0].TransactionID != txs[0].ID {
		t.Fatalf("warning page scope: err=%v total=%d rows=%v", err, total, warnings)
	}
	rows, total, err := txRepo.ListTransactions(ctx, "alle", "%", "date", "desc", 0, 50)
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("literal search: err=%v total=%d rows=%v", err, total, rows)
	}
	rows, total, err = txRepo.ListTransactions(ctx, "alle", "", "invalid SQL", "invalid", 0, 1)
	if err != nil || total != 105 || len(rows) != 1 || rows[0].ID != txs[104].ID {
		t.Fatalf("sort whitelist fallback: err=%v total=%d rows=%v", err, total, rows)
	}
	if _, err := testDB.Exec("UPDATE fees.bank_transactions SET is_hidden = true WHERE id IN ($1, $2)",
		txs[103].ID, txs[104].ID); err != nil {
		t.Fatal(err)
	}
	rows, total, err = txRepo.ListTransactions(ctx, "alle", "", "amount", "desc", 0, 200)
	if err != nil || total != 104 || len(rows) != 104 || rows[0].ID != txs[104].ID {
		t.Fatalf("hidden warning union: err=%v total=%d rows=%v", err, total, rows)
	}

}
