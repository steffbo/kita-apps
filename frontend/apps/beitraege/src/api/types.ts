// API types for the fees backend.
//
// Response shapes are derived from the generated OpenAPI schema
// (src/api/schema.d.ts, produced by `bun run generate:api` from
// openapi/fees/openapi3.yaml) so field names/types can no longer drift
// from the backend contract.
//
// swag runs with --requiredByDefault: every JSON field is required unless the
// Go struct tag says `binding:"optional"`, which backend-fees sets on all
// `omitempty` and pointer fields. So the generated optionality mirrors what
// the API actually sends; a few fields are narrowed or widened (`| null`)
// below where the UI relies on it. Request payloads stay hand-written because
// the backend validates them manually.
import type { components } from './schema';

type Schema = components['schemas'];

// ── Auth ─────────────────────────────────────────────────────────────────────
export type LoginRequest = Schema['LoginRequest'];
// The refresh token is never visible to scripts: the backend keeps it in the
// httpOnly cookie `fees_refresh`. Responses only carry the access token.
export type LoginResponse = Schema['LoginResponse'];
export type RefreshResponse = Schema['RefreshResponse'];
export type User = Schema['User'];
export type ChangePasswordRequest = Schema['ChangePasswordRequest'];

// ── User management (admin) ──────────────────────────────────────────────────
export type UserAccount = Schema['UserAccount'];
export type UserRole = UserAccount['role'];
export type UserAccountRequest = Schema['UserAccountRequest'];
export type CreateUserAccountRequest = Schema['CreateUserAccountRequest'];

// ── Reminder settings and runs ───────────────────────────────────────────────
export type ReminderPaymentSettings = Schema['handler.ReminderPaymentSettingsPayload'];
export type ReminderSettingsResponse = 
  Schema['ReminderSettingsResponse'] & {
    payment: ReminderPaymentSettings;
  }
;
export interface UpdateReminderSettingsRequest {
  autoEnabled: boolean;
  payment?: ReminderPaymentSettings;
}
export type ReminderRunStage = NonNullable<Schema['ReminderRunResponse']['stage']>;
export type ReminderWarning = Schema['ReminderWarningResponse'];
export type ReminderPreview = Schema['ReminderPreviewResponse'];
export interface ReminderRunOverride {
  subject?: string;
  body?: string;
}
export interface ReminderRunBody {
  includeQR?: boolean;
  overrides?: Record<string, ReminderRunOverride>;
}
export type ReminderRunResponse = Omit<Schema['ReminderRunResponse'], 'warnings' | 'previews'> & {
  warnings?: ReminderWarning[];
  previews?: ReminderPreview[];
};

// ── Family reminder cases ────────────────────────────────────────────────────
export type ReminderCaseFeeStatus = NonNullable<Schema['service.ReminderCaseFee']['status']>;
export type ReminderCaseStage = 'initial' | 'final';
export type ReminderCaseFee = Schema['service.ReminderCaseFee'];
export type ReminderCase = Schema['service.ReminderCase'];
export type ReminderCasesResult = Schema['service.ReminderCasesResult'];
export type ReminderCasePlannedFee = Schema['service.ReminderCasePlannedFee'];
export type ReminderCasePreview = Schema['service.ReminderCasePreview'];
export type ReminderCaseSendResult = Schema['service.ReminderCaseSendResult'];
export interface ReminderCaseRequest {
  stage: ReminderCaseStage;
  runDate?: string;
  feeIds: string[];
  includeQR?: boolean;
  subject?: string;
  body?: string;
  previewedAt?: string;
}
export interface ReminderCaseConflict {
  message: string;
  feeIds: string[];
}
export class ReminderCaseConflictError extends Error {
  feeIds: string[];
  constructor(conflict: ReminderCaseConflict) {
    super(conflict.message);
    this.name = 'ReminderCaseConflictError';
    this.feeIds = conflict.feeIds ?? [];
  }
}

export type EmailLogType =
  | 'REMINDER_INITIAL'
  | 'REMINDER_FINAL'
  | 'MEMBERSHIP_REMINDER_INITIAL'
  | 'MEMBERSHIP_REMINDER_FINAL'
  | 'PASSWORD_RESET'
  | string;

export interface EmailLog {
  id: string;
  sentAt: string;
  toEmail: string;
  subject: string;
  body?: string | null;
  emailType: EmailLogType;
  sentBy?: string | null;
}

// ── Children ─────────────────────────────────────────────────────────────────
export type Child = Omit<Schema['domain.Child'], 'legalHours' | 'careHours'> & {
  legalHours?: number | null;
  careHours?: number | null;
};
export type NextMemberNumberResponse = Schema['NextMemberNumberResponse'];
export interface CreateChildRequest {
  memberNumber: string;
  firstName: string;
  lastName: string;
  birthDate: string;
  entryDate: string;
  exitDate?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  legalHours?: number | null;
  legalHoursUntil?: string;
  careHours?: number | null;
}
export interface UpdateChildRequest {
  firstName?: string;
  lastName?: string;
  birthDate?: string;
  entryDate?: string;
  exitDate?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  legalHours?: number | null;
  legalHoursUntil?: string;
  careHours?: number | null;
  isActive?: boolean;
  householdId?: string;
}
export type CareHoursHistoryEntry = Omit<
  Schema['CareHoursHistoryEntry'],
  'careHours' | 'effectiveUntil'
> & {
  careHours?: number | null;
  effectiveUntil?: string | null;
};
export interface CreateCareHoursHistoryRequest {
  careHours?: number | null;
  validFrom: string;
}
export type LegalHoursHistoryEntry = Omit<
  Schema['LegalHoursHistoryEntry'],
  'legalHours' | 'effectiveUntil'
> & {
  legalHours?: number | null;
  effectiveUntil?: string | null;
};
export interface CreateLegalHoursHistoryRequest {
  legalHours?: number | null;
  validFrom: string;
}

// ── Notes ────────────────────────────────────────────────────────────────────
// Free-text notes for children. Every authenticated user may edit and delete
// any note; there is no owner field.
export type ChildNote = Schema['ChildNote'];
export interface CreateChildNoteRequest {
  text: string;
}
export interface UpdateChildNoteRequest {
  text: string;
}

// ── Households ───────────────────────────────────────────────────────────────
export type IncomeStatus = Schema['domain.IncomeStatus'] | '';
export type Household = Schema['domain.Household'];
export interface CreateHouseholdRequest {
  name: string;
  annualHouseholdIncome?: number;
  incomeStatus?: IncomeStatus;
}
export interface UpdateHouseholdRequest {
  name?: string;
  annualHouseholdIncome?: number;
  incomeStatus?: IncomeStatus;
  childrenCountForFees?: number;
}

// ── Members (Vereinsmitglieder - can exist independently of children) ───────
export type Member = Schema['domain.Member'];
export interface CreateMemberRequest {
  memberNumber?: string; // Auto-generated if not provided
  firstName: string;
  lastName: string;
  email?: string;
  phone?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  householdId?: string;
  membershipStart: string;
  membershipEnd?: string;
}
export interface UpdateMemberRequest {
  firstName?: string;
  lastName?: string;
  email?: string;
  phone?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  householdId?: string;
  membershipStart?: string;
  membershipEnd?: string;
  isActive?: boolean;
}

// ── Parents ──────────────────────────────────────────────────────────────────
export type Parent = Schema['domain.Parent'];
export interface CreateParentRequest {
  firstName: string;
  lastName: string;
  birthDate?: string;
  email?: string;
  phone?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  householdId?: string;
  memberId?: string; // Link to existing Member
}
export interface UpdateParentRequest {
  firstName?: string;
  lastName?: string;
  birthDate?: string;
  email?: string;
  phone?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  householdId?: string;
  memberId?: string; // Link to existing Member
}

// ── Fees ─────────────────────────────────────────────────────────────────────
export type FeeType = Schema['domain.FeeType'];
export type FeeStatus = 'OPEN' | 'PAID' | 'OVERDUE';

export type FeeExpectation = Schema['domain.FeeExpectation'];
export type PaymentMatch = Schema['domain.PaymentMatch'];
export type FeeOverview = Schema['domain.FeeOverview'];
export type MonthSummary = Schema['domain.MonthSummary'];
export interface GenerateFeeRequest {
  year: number;
  month?: number;
}
export interface GenerateFeeResult {
  created: number;
  skipped: number;
}
export interface CreateFeeRequest {
  childId: string;
  feeType: FeeType;
  year: number;
  month?: number;
  amount?: number;
  dueDate?: string;
  reconciliationYear?: number;
}

// ── Bank Transactions ────────────────────────────────────────────────────────
export type BankTransaction = Schema['domain.BankTransaction'];
export type MatchSuggestion = Omit<Schema['domain.MatchSuggestion'], 'transaction'> & {
  transaction: BankTransaction;
};
/** A bank CSV row that could not be saved or processed during import/rescan. */
export type ImportError = Schema['ImportError'];
export type ImportResult = Omit<Schema['service.ImportResult'], 'suggestions' | 'errors'> & {
  suggestions: MatchSuggestion[];
  errors?: ImportError[];
};
export interface MatchConfirmation {
  transactionId: string;
  expectationId: string;
}
export type ConfirmResult = Schema['ConfirmMatchResponse'];
export type ImportBatch = Omit<Schema['domain.ImportBatch'], 'errors'> & {
  errors?: ImportError[];
};

export type BankingSyncStatusType =
  | 'idle'
  | 'running'
  | 'waiting_for_2fa'
  | 'success'
  | 'error'
  | 'cancelled';
export interface BankingSyncStatus {
  status: BankingSyncStatusType;
  runId?: string | null;
  startedAt?: string | null;
  finishedAt?: string | null;
  lastError?: string | null;
  lastMessage?: string | null;
  downloadPath?: string | null;
  uploadResult?: unknown;
  logs?: string[];
  updatedAt?: string;
}

// ── Pagination envelope (response.Paginated; not generic in the spec) ────────
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  perPage: number;
  totalPages: number;
}

// ── API Error ────────────────────────────────────────────────────────────────
export interface ApiError {
  error: string;
  message?: string;
}

// ── Known IBANs (IBAN Learning System) ───────────────────────────────────────
export type KnownIBANStatus = Schema['domain.KnownIBANStatus'];
export type KnownIBAN = Schema['domain.KnownIBAN'];
export type KnownIBANSummary = Schema['ChildTrustedIBANsResponse'];
export type RescanResult = Omit<Schema['RescanResponse'], 'errors'> & {
  errors?: ImportError[];
};
export type DismissResult = Schema['DismissTransactionResponse'];
export type HideResult = Schema['HideTransactionResponse'];
export type UnmatchResult = Schema['UnmatchTransactionResponse'];
export type ChildUnmatchedSuggestionsResponse = Omit<Schema['ChildUnmatchedSuggestionsResponse'], 'suggestions'> & {
  suggestions: MatchSuggestion[];
};
export interface AllocationInput {
  expectationId: string;
  amount: number;
}
export type AllocateTransactionResult = Schema['AllocateTransactionResponse'];

// ── Transaction Warnings ─────────────────────────────────────────────────────
export type WarningType = Schema['domain.WarningType'];
export type ResolutionType = Schema['domain.ResolutionType'];
export type TransactionWarning = Omit<Schema['domain.TransactionWarning'], 'expectedAmount' | 'actualAmount'> & {
  expectedAmount?: number | null;
  actualAmount?: number | null;
};
export type ResolveLateFeeResult = Schema['ResolveLateFeeResponse'];

// ── Child Import ─────────────────────────────────────────────────────────────
export type ChildImportParseResult = Schema['service.ChildImportParseResult'];
export interface ChildImportPreviewRequest {
  fileContent: string; // Base64 encoded CSV content
  separator: string;
  mapping: Record<string, number>; // systemField -> csvColumnIndex
  skipHeader: boolean;
}
// NOTE: the preview/row family below is kept as plain interfaces: they are
// also sent back in execute requests and carry client-side extras.
export interface ChildPreview {
  memberNumber: string;
  firstName: string;
  lastName: string;
  birthDate: string;
  entryDate: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  legalHours?: number | null;
  careHours?: number | null;
}
export interface ParentMatch {
  id: string;
  firstName: string;
  lastName: string;
  email?: string;
}
export interface ParentPreview {
  firstName: string;
  lastName: string;
  email?: string;
  phone?: string;
  existingMatches?: ParentMatch[];
  alreadyLinked?: boolean; // True if already linked to the existing child
  linkedParentId?: string; // ID of the already linked parent
}
export interface FieldConflict {
  field: string;
  fieldLabel: string;
  existingValue: string;
  newValue: string;
}
export interface ChildImportPreviewRow {
  index: number;
  child: ChildPreview;
  parent1?: ParentPreview;
  parent2?: ParentPreview;
  warnings: string[];
  isDuplicate: boolean;
  existingChildId?: string;
  existingChild?: ChildPreview; // The existing child data for comparison when duplicate
  action: 'create' | 'update' | 'no_change'; // What action will be taken
  fieldConflicts?: FieldConflict[]; // Conflicts between CSV and existing data
  isValid: boolean;
}
export interface ChildImportPreviewResult {
  rows: ChildImportPreviewRow[];
  validCount: number;
  errorCount: number;
}
export interface ChildImportRow {
  index: number;
  child: ChildPreview;
  parent1?: ParentPreview;
  parent2?: ParentPreview;
  existingChildId?: string; // If set, this is a merge/update operation
  mergeParents?: boolean; // If true, add parents to existing child
  fieldUpdates?: Record<string, string>; // Field -> value for updates (from conflict resolution)
}
export interface ParentDecision {
  rowIndex: number;
  parentIndex: 1 | 2;
  action: 'create' | 'link';
  existingParentId?: string;
}
export interface ChildImportExecuteRequest {
  rows: ChildImportRow[];
  parentDecisions: ParentDecision[];
}
export interface ChildImportError {
  rowIndex: number;
  error: string;
}
export interface ChildImportExecuteResult {
  childrenCreated?: number;
  childrenUpdated?: number;
  parentsCreated?: number;
  parentsLinked?: number;
  errors: ChildImportError[];
}

// System fields for mapping UI (client-side only)
export interface SystemField {
  key: string;
  label: string;
  required: boolean;
  group: 'child' | 'parent1' | 'parent2';
}

// ── Childcare Fee Calculation ────────────────────────────────────────────────
export type ChildAgeType = 'krippe' | 'kindergarten';

export interface ChildcareFeeInput {
  childAgeType?: ChildAgeType;
  income: number;
  siblingsCount?: number;
  careHours?: number;
  highestRate?: boolean;
  fosterFamily?: boolean;
}

export type ChildcareFeeResult = Schema['domain.ChildcareFeeResult'];

// ── Ledger ───────────────────────────────────────────────────────────────────
export type LedgerEntry = Schema['LedgerEntry'];
export type LedgerSummary = Schema['LedgerSummary'];
export type ChildLedger = Schema['ChildLedger'];

// ── Stichtagsmeldung ─────────────────────────────────────────────────────────
export type StichtagsmeldungStats = Schema['StichtagsmeldungStats'];
export type MemberCountAsOf = Schema['MemberCountAsOf'];
export type StichtagsmeldungReport = Schema['StichtagsmeldungReport'];
export type U3IncomeBreakdown = Schema['U3IncomeBreakdown'];
export type CareHoursBreakdownItem = Omit<
  Schema['CareHoursBreakdownItem'],
  'careHours'
> & {
  careHours?: number | null;
};
export type LegalHoursBreakdownItem = Omit<
  Schema['LegalHoursBreakdownItem'],
  'legalHours'
> & {
  legalHours?: number | null;
};
export type U3ChildDetail = Omit<Schema['handler.U3ChildDetailResponse'], 'householdIncome' | 'incomeStatus'> & {
  householdIncome: number | null;
  incomeStatus: string | null;
};

// ── Fee Coverage Timeline ────────────────────────────────────────────────────
export type CoverageStatus = NonNullable<Schema['handler.FeeCoverageResponse']['status']>;
export type CoveredTransaction = Schema['handler.CoveredTransactionResponse'];
export type FeeCoverage = Schema['handler.FeeCoverageResponse'];

// ── Einstufung (Fee Classification) ──────────────────────────────────────────
export interface IncomeDetails {
  grossIncome: number;
  socialSecurityShare: number;
  privateInsurance: number;
  tax: number;
  advertisingCosts: number;
  minijobIncome: number;
  unemploymentBenefit: number;
  capitalIncome: number;
  rentalIncome: number;
  otherIncome: number;
  profit: number;
  welfareExpense: number;
  selfEmployedTax: number;
  parentalBenefit: number;
  parentalBenefitPlus: number;
  maternityBenefit: number;
  insurances: number;
  maintenanceToPay: number;
  maintenanceReceived: number;
}
export interface HouseholdIncomeCalculation {
  parent1: IncomeDetails;
  parent2: IncomeDetails;
}
export type EinstufungMonthRow = Schema['domain.EinstufungMonthRow'];
export type Einstufung = Omit<Schema['Einstufung'], 'incomeCalculation' | 'monthlyTable' | 'child' | 'household'> & {
  incomeCalculation: HouseholdIncomeCalculation;
  monthlyTable?: EinstufungMonthRow[];
  child?: Child;
  household?: Household;
};
export interface CreateEinstufungRequest {
  childId: string;
  year: number;
  validFrom: string;
  incomeCalculation: HouseholdIncomeCalculation;
  highestRateVoluntary: boolean;
  careHoursPerWeek: number;
  childrenCount: number;
  notes?: string;
}
export interface UpdateEinstufungRequest {
  incomeCalculation?: HouseholdIncomeCalculation;
  highestRateVoluntary?: boolean;
  careHoursPerWeek?: number;
  childrenCount?: number;
  validFrom?: string;
  notes?: string;
}
export interface CreateFollowUpEinstufungRequest {
  changeDate: string;
  incomeCalculation: HouseholdIncomeCalculation;
  highestRateVoluntary: boolean;
  careHoursPerWeek: number;
  childrenCount: number;
  notes?: string;
}
export type CreditReviewPeriod = Schema['CreditReviewPeriod'];
export type ChildcareExpectationSyncResult = Schema['ChildcareExpectationSyncResult'];
export type CreateFollowUpEinstufungResponse = Omit<Schema['CreateFollowUpEinstufungResponse'], 'einstufung' | 'expectationChanges'> & {
  einstufung: Einstufung;
  expectationChanges: ChildcareExpectationSyncResult;
};
export type CalculateIncomeResponse = Schema['CalculateIncomeResponse'];

// ── Fee schedules (Beitragsordnung) ──────────────────────────────────────────
export type FeeTableRow = Schema['FeeTableRow'];
/** `kindergartenTable` (Ü3) is reference only and may be missing. */
export type FeeScheduleConfig = Schema['FeeScheduleConfig'];
export type FeeScheduleStatus = NonNullable<Schema['FeeScheduleVersion']['status']>;
/** One version of the fee regulation; only `planned` versions are editable. */
export type FeeScheduleVersion = Schema['FeeScheduleVersion'];
export interface FeeScheduleRequest {
  validFrom: string;
  name: string;
  config: FeeScheduleConfig;
}
