package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/email"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/letter"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// LetterEmailSender is satisfied by the SMTP service.
type LetterEmailSender interface {
	IsEnabled() bool
	SendTextEmailWithAttachments(to []string, subject, body string, attachments []email.Attachment) error
}

// BoardTermLister lists the club's board appointments.
type BoardTermLister interface {
	ListTerms(ctx context.Context) ([]domain.BoardTerm, error)
}

// EinstufungLetterService renders the Einstufung letter and emails it to the parents.
type EinstufungLetterService struct {
	einstufungs    *EinstufungService
	einstufungRepo repository.EinstufungRepository
	board          BoardTermLister
	settings       repository.SettingsRepository
	logs           repository.EmailLogRepository
	sender         LetterEmailSender
}

// NewEinstufungLetterService creates the letter service.
func NewEinstufungLetterService(
	einstufungs *EinstufungService,
	einstufungRepo repository.EinstufungRepository,
	board BoardTermLister,
	settings repository.SettingsRepository,
	logs repository.EmailLogRepository,
	sender LetterEmailSender,
) *EinstufungLetterService {
	return &EinstufungLetterService{einstufungs, einstufungRepo, board, settings, logs, sender}
}

// LetterFile is a rendered letter.
type LetterFile struct {
	Filename string
	Data     []byte
}

// SendEinstufungLetterInput is the email text written in the UI.
type SendEinstufungLetterInput struct {
	Subject string
	Body    string
}

// SendEinstufungLetterResult describes a sent letter.
type SendEinstufungLetterResult struct {
	Recipients []string
	SentAt     time.Time
	Filename   string
}

// bankNames maps known BICs to the bank name printed in the letter footer.
var bankNames = map[string]string{"BFSWDE33XXX": "Bank für Sozialwirtschaft AG"}

// boardOrder is the order of offices in the letter footer; others follow alphabetically.
var boardOrder = []string{"Vorsitz", "Bauliches", "Finanzen", "Elternarbeit", "Schriftführung", "Personal"}

// PDF renders the letter for an Einstufung.
func (s *EinstufungLetterService) PDF(ctx context.Context, id uuid.UUID) (*LetterFile, error) {
	e, err := s.einstufungs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.render(ctx, e, letterDate(e))
}

// Send emails the letter as PDF attachment to the household's parents and logs it.
func (s *EinstufungLetterService) Send(ctx context.Context, id, actorID uuid.UUID,
	input SendEinstufungLetterInput) (*SendEinstufungLetterResult, error) {
	subject, body := strings.TrimSpace(input.Subject), strings.TrimSpace(input.Body)
	if subject == "" || body == "" {
		return nil, newValidationError("Betreff und Text der E-Mail dürfen nicht leer sein.")
	}
	if s.sender == nil || !s.sender.IsEnabled() {
		return nil, ErrEmailDisabled
	}
	e, err := s.einstufungs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	recipients := parentEmails(e.Household)
	if len(recipients) == 0 {
		return nil, newValidationError("Bei den Eltern ist keine E-Mail-Adresse hinterlegt.")
	}
	sentAt := util.Now()
	file, err := s.render(ctx, e, util.DateOf(sentAt))
	if err != nil {
		return nil, err
	}
	if err := s.sender.SendTextEmailWithAttachments(recipients, subject, body, []email.Attachment{
		{Filename: file.Filename, ContentType: "application/pdf", Data: file.Data},
	}); err != nil {
		return nil, fmt.Errorf("E-Mail konnte nicht gesendet werden: %w", err)
	}

	// The mail is out; failures below must not invite a second send.
	result := &SendEinstufungLetterResult{Recipients: recipients, SentAt: sentAt, Filename: file.Filename}
	if err := s.einstufungRepo.MarkLetterSent(ctx, e.ID, sentAt); err != nil {
		return result, fmt.Errorf("E-Mail gesendet, Versanddatum aber nicht gespeichert: %w", err)
	}
	payload, _ := json.Marshal(map[string]string{"einstufungId": e.ID.String(), "attachment": file.Filename})
	raw := json.RawMessage(payload)
	householdID := e.HouseholdID
	if err := s.logs.Create(ctx, &domain.EmailLog{
		ID: uuid.New(), SentAt: sentAt, ToEmail: strings.Join(recipients, ", "), Subject: subject,
		Body: &body, EmailType: domain.EmailLogTypeEinstufung, Payload: &raw,
		SentBy: &actorID, HouseholdID: &householdID,
	}); err != nil {
		return result, fmt.Errorf("E-Mail gesendet, Protokoll aber nicht gespeichert: %w", err)
	}
	return result, nil
}

// letterDate is the send date while the Einstufung is unchanged since sending, else today.
func letterDate(e *domain.Einstufung) time.Time {
	if e.LetterSentAt != nil && !e.UpdatedAt.After(*e.LetterSentAt) {
		return util.DateOf(e.LetterSentAt.In(util.Berlin))
	}
	return util.Today()
}

func (s *EinstufungLetterService) render(ctx context.Context, e *domain.Einstufung,
	date time.Time) (*LetterFile, error) {
	if e.Child == nil {
		return nil, ErrNotFound
	}
	var previous *domain.Einstufung
	if e.SourceEinstufungID != nil {
		source, err := s.einstufungs.GetByID(ctx, *e.SourceEinstufungID)
		if err != nil {
			return nil, fmt.Errorf("load previous Einstufung: %w", err)
		}
		previous = source
	}
	board, err := s.boardAt(ctx, date)
	if err != nil {
		return nil, err
	}
	bank, err := loadPaymentSettings(ctx, s.settings)
	if err != nil {
		return nil, err
	}
	data := buildEinstufungLetter(e, previous, date)
	data.Board = board
	data.Bank = letter.BankAccount{
		Holder: bank.RecipientName, IBAN: formatIBAN(bank.IBAN), BIC: bank.BIC, Bank: bankNames[bank.BIC],
	}
	pdf, err := letter.RenderEinstufung(data)
	if err != nil {
		return nil, err
	}
	return &LetterFile{Filename: einstufungFilename(e), Data: pdf}, nil
}

// buildEinstufungLetter maps an Einstufung to the letter. The wording follows
// EinstufungPDF.vue.
func buildEinstufungLetter(e, previous *domain.Einstufung, date time.Time) letter.EinstufungLetter {
	start := e.EffectiveFromMonth
	if start.IsZero() {
		start = e.ValidFrom
	}
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)

	rows := make([]letter.MonthRow, 0, len(e.MonthlyTable))
	appendRows := func(source *domain.Einstufung, isPrevious bool) {
		for _, row := range source.MonthlyTable {
			rows = append(rows, letter.MonthRow{
				Month:    time.Date(row.Year, time.Month(row.Month), 1, 0, 0, 0, 0, time.UTC),
				CareType: row.CareType, CareHours: row.CareHoursPerWeek,
				ChildcareFee: row.ChildcareFee, FoodFee: row.FoodFee, Previous: isPrevious,
			})
		}
	}
	if previous != nil {
		appendRows(previous, true)
	}
	appendRows(e, false)
	if len(e.MonthlyTable) == 0 {
		careType := "Krippe"
		if e.CareType == domain.ChildAgeTypeKindergarten {
			careType = "Kindergarten"
		}
		rows = append(rows, letter.MonthRow{Month: start, CareType: careType, CareHours: e.CareHoursPerWeek,
			ChildcareFee: e.MonthlyChildcareFee, FoodFee: e.MonthlyFoodFee})
	}

	l := letter.EinstufungLetter{
		LetterDate:  date,
		Year:        e.Year,
		ValidFrom:   start,
		Provisional: e.Provisional,
		BasisText:   letterBasisText(e),
		Periods:     letter.BuildFeePeriods(rows, e.ValidUntil == nil),
	}
	if e.Child != nil {
		l.ChildName = e.Child.FirstName + " " + e.Child.LastName
		l.MemberNumber = e.Child.MemberNumber
		l.BirthDate = e.Child.BirthDate
		l.EntryDate = e.Child.EntryDate
	}
	// A follow-up carries no membership fee of its own; quote the one already paid.
	paid := e.AnnualMembershipFee
	if previous != nil && paid == 0 {
		paid = previous.AnnualMembershipFee
	}
	l.MembershipFee = paid
	fee := letter.FormatEuro(e.AnnualMembershipFee)
	switch {
	case e.SourceEinstufungID != nil && paid > 0:
		l.MembershipNote = "Vereinsbeitrag: " + letter.FormatEuro(paid) + " jährlich – für dieses Jahr bereits " +
			"bezahlt, in den oben genannten Monatsbeträgen nicht enthalten."
	case e.SourceEinstufungID != nil:
	case e.AnnualMembershipFee > 0:
		l.MembershipNote = "Vereinsbeitrag: " + fee + " jährlich – einmalig fällig mit dem ersten Beitrag im " +
			letter.FormatMonthShort(start) + ", in den Monatsbeträgen nicht enthalten."
	}
	return l
}

func letterBasisText(e *domain.Einstufung) string {
	provisional := ""
	if e.Provisional {
		provisional = " Die Einstufung ist vorläufig, bis die fehlenden Nachweise vorliegen."
	}
	switch {
	case e.IncomeBasis == domain.IncomeBasisHighestRate:
		return "Die Einstufung erfolgte aufgrund der freiwilligen Anerkennung des Höchstsatzes." + provisional
	case e.IncomeBasis == domain.IncomeBasisNotRequired:
		return "Die Betreuung im Kindergarten ist beitragsfrei; Einkommensnachweise sind dafür nicht erforderlich." +
			provisional
	case strings.HasPrefix(e.FeeRule, "Beitragsfrei"):
		return "Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise vorgenommen. " +
			"Gemäß Elternentlastungsgesetz ist der Beitrag beitragsfrei." + provisional +
			" Bei Änderungen informiert uns bitte umgehend."
	}
	rule := "nach der Elternbeitragssatzung"
	if strings.Contains(e.FeeRule, "Entlastung") {
		rule = "nach dem Elternentlastungsgesetz 2023/2024"
	}
	return "Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise und " + rule +
		" vorgenommen." + provisional + " Bei Änderungen informiert uns bitte umgehend."
}

// boardAt returns the board members in office at date, in footer order.
func (s *EinstufungLetterService) boardAt(ctx context.Context, date time.Time) ([]letter.BoardMember, error) {
	terms, err := s.board.ListTerms(ctx)
	if err != nil {
		return nil, fmt.Errorf("load board terms: %w", err)
	}
	rank := func(office string) int {
		for i, o := range boardOrder {
			if o == office {
				return i
			}
		}
		return len(boardOrder)
	}
	active := make([]domain.BoardTerm, 0, len(terms))
	for _, t := range terms {
		if !t.StartDate.After(date) && (t.EndDate == nil || !t.EndDate.Before(date)) {
			active = append(active, t)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		ri, rj := rank(active[i].Office), rank(active[j].Office)
		if ri != rj {
			return ri < rj
		}
		if active[i].Office != active[j].Office {
			return active[i].Office < active[j].Office
		}
		return active[i].MemberName < active[j].MemberName
	})
	members := make([]letter.BoardMember, len(active))
	for i, t := range active {
		members[i] = letter.BoardMember{Name: t.MemberName, Role: t.Office}
	}
	return members, nil
}

func parentEmails(household *domain.Household) []string {
	if household == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range household.Parents {
		if p.Email == nil {
			continue
		}
		address := strings.TrimSpace(*p.Email)
		key := strings.ToLower(address)
		if address == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, address)
	}
	return out
}

// einstufungFilename is "Einstufung_<Jahr>_<Nachname>_<Vorname>.pdf".
func einstufungFilename(e *domain.Einstufung) string {
	clean := strings.NewReplacer("/", "-", "\\", "-", "\"", "", ":", "-")
	parts := []string{"Einstufung", fmt.Sprint(e.Year)}
	if e.Child != nil {
		parts = append(parts, strings.Join(strings.Fields(e.Child.LastName), "_"),
			strings.Join(strings.Fields(e.Child.FirstName), "_"))
	}
	return clean.Replace(strings.Join(parts, "_")) + ".pdf"
}

// formatIBAN groups an IBAN in blocks of four.
func formatIBAN(iban string) string {
	compact := normalizeIBAN(iban)
	var groups []string
	for len(compact) > 4 {
		groups = append(groups, compact[:4])
		compact = compact[4:]
	}
	return strings.Join(append(groups, compact), " ")
}
