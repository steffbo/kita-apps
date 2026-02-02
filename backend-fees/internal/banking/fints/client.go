package fints

import (
	"context"
	"fmt"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	hbciClient "github.com/mitch000001/go-hbci/client"
	hbciDomain "github.com/mitch000001/go-hbci/domain"
)

// Client represents a FinTS/HBCI client for fetching bank transactions.
type Client struct {
	config Config
}

// Config contains the FinTS connection parameters.
type Config struct {
	BankCode      string // Bankleitzahl (e.g., "37020500" for SozialBank)
	UserID        string // Online banking username (NetKey for SozialBank)
	PIN           string // PIN or password
	FinTSURL      string // FinTS endpoint URL
	AccountNumber string // Optional: specific account number to fetch
}

// NewClient creates a new FinTS client.
func NewClient(config Config) *Client {
	return &Client{
		config: config,
	}
}

// createHBCIClient creates the underlying go-hbci client.
func (c *Client) createHBCIClient() (*hbciClient.Client, error) {
	config := hbciClient.Config{
		BankID:      c.config.BankCode,
		AccountID:   c.config.UserID, // go-hbci uses AccountID for the login username
		PIN:         c.config.PIN,
		URL:         c.config.FinTSURL,
		HBCIVersion: 300, // FinTS 3.0 - required for banks not in the built-in database
	}

	return hbciClient.New(config)
}

// FetchTransactions retrieves transactions from the bank account.
// For transactions within the last 90 days, no TAN should be required.
func (c *Client) FetchTransactions(ctx context.Context, since time.Time) ([]domain.BankTransaction, error) {
	client, err := c.createHBCIClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HBCI client: %w", err)
	}

	// Get accounts
	accounts, err := client.Accounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found")
	}

	// Find the target account
	var targetAccount hbciDomain.AccountConnection
	if c.config.AccountNumber != "" {
		found := false
		for _, acc := range accounts {
			// AccountInformation has AccountConnection which contains AccountID
			if acc.AccountConnection.AccountID == c.config.AccountNumber {
				targetAccount = acc.AccountConnection
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("account %s not found", c.config.AccountNumber)
		}
	} else {
		// Use first account
		targetAccount = accounts[0].AccountConnection
	}

	// Define timeframe using ShortDate
	startDate := hbciDomain.NewShortDate(since)
	endDate := hbciDomain.NewShortDate(time.Now())
	timeframe := hbciDomain.TimeframeFromDate(startDate)
	// Note: Timeframe only supports single date in this version
	// For range queries, we might need to adjust
	_ = endDate

	// Fetch transactions
	// For recent transactions (within 90 days), this should work without TAN
	transactions, err := client.AccountTransactions(targetAccount, timeframe, false, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Convert HBCI transactions to domain transactions
	var result []domain.BankTransaction
	for _, tx := range transactions {
		converted := c.convertTransaction(&tx)
		result = append(result, converted)
	}

	return result, nil
}

// TestConnection tests the connection to the bank.
func (c *Client) TestConnection(ctx context.Context) error {
	client, err := c.createHBCIClient()
	if err != nil {
		return fmt.Errorf("failed to create HBCI client: %w", err)
	}

	// Try to get accounts - this tests the connection
	_, err = client.Accounts()
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// convertTransaction converts an HBCI transaction to domain.BankTransaction.
func (c *Client) convertTransaction(tx *hbciDomain.AccountTransaction) domain.BankTransaction {
	// Combine purpose fields for description
	description := tx.Purpose
	if tx.Purpose2 != "" {
		description += " " + tx.Purpose2
	}

	// Get payer name from Name field
	payerName := tx.Name

	return domain.BankTransaction{
		BookingDate: tx.BookingDate,
		ValueDate:   tx.ValutaDate,
		PayerName:   &payerName,
		PayerIBAN:   nil, // IBAN not directly available in AccountTransaction, would need SepaAccountTransactions
		Description: &description,
		Amount:      tx.Amount.Amount,
		Currency:    tx.Amount.Currency,
	}
}
