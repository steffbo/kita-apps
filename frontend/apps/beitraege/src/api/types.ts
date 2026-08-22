// API types for the fees backend.
//
// Response shapes are derived from the generated OpenAPI schema
// (src/api/schema.d.ts, produced by `bun run generate:api` from
// openapi/fees/openapi3.yaml) so field names/types can no longer drift
// from the backend contract.
//
// The Go swag annotations declare no `required` markers, so every generated
// property is optional — including fields the API always sends. DeepStrict
// restores non-optionality; fields that are genuinely sparse (`omitempty`,
// joined/computed values) are re-loosened per type below, mirroring the
// runtime behaviour. Request payloads stay hand-written because the backend
// validates them manually (spec marks everything optional there too).
import type { components } from './schema';

type Schema = components['schemas'];

/** Recursively strip optionality introduced by swag's missing `required` markers. */
type DeepStrict<T> =
  T extends (infer U)[]
    ? DeepStrict<U>[]
    : T extends object
      ? { [K in keyof T]-?: DeepStrict<T[K]> }
      : T;

/** Re-declare selected keys as optional (for `omitempty` / joined fields). */
type Loose<T, K extends keyof T> = Omit<T, K> & Partial<{ [P in K]: T[P] }>;

// ── Auth ─────────────────────────────────────────────────────────────────────
export type LoginRequest = Schema['LoginRequest'];
export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
}
export type User = DeepStrict<Schema['User']>;

// ── Reminder settings and runs ───────────────────────────────────────────────
export type ReminderPaymentSettings = Loose<
  DeepStrict<Schema['handler.ReminderPaymentSettingsPayload']>,
  'bic'
>;
export type ReminderSettingsResponse = DeepStrict<
  Schema['ReminderSettingsResponse'] & {
    payment: ReminderPaymentSettings;
  }
>;
export interface UpdateReminderSettingsRequest {
  autoEnabled: boolean;
  payment?: ReminderPaymentSettings;
}
export type ReminderRunStage = NonNullable<Schema['ReminderRunResponse']['stage']>;
export type ReminderWarning = DeepStrict<Schema['ReminderWarningResponse']>;
export type ReminderPreview = Loose<
  DeepStrict<Schema['ReminderPreviewResponse']>,
  'qrImageDataUrl' | 'qrPayload'
>;
export interface ReminderRunOverride {
  subject?: string;
  body?: string;
}
export interface ReminderRunBody {
  includeQR?: boolean;
  overrides?: Record<string, ReminderRunOverride>;
}
export type ReminderRunResponse = Loose<
  DeepStrict<Schema['ReminderRunResponse']>,
  'warnings' | 'previews' | 'message' | 'recipient' | 'reminderCreated'
> & {
  warnings?: ReminderWarning[];
  previews?: ReminderPreview[];
};

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
export type Child = Loose<
  Omit<DeepStrict<Schema['domain.Child']>, 'legalHours' | 'careHours'>,
  | 'householdId'
  | 'exitDate'
  | 'street'
  | 'streetNo'
  | 'postalCode'
  | 'city'
  | 'legalHoursUntil'
  | 'parents'
  | 'household'
  | 'openFeesCount'
> & {
  legalHours?: number | null;
  careHours?: number | null;
};
export type NextMemberNumberResponse = DeepStrict<Schema['NextMemberNumberResponse']>;
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
  DeepStrict<Schema['CareHoursHistoryEntry']>,
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
  DeepStrict<Schema['LegalHoursHistoryEntry']>,
  'legalHours' | 'effectiveUntil'
> & {
  legalHours?: number | null;
  effectiveUntil?: string | null;
};
export interface CreateLegalHoursHistoryRequest {
  legalHours?: number | null;
  validFrom: string;
}

// ── Households ───────────────────────────────────────────────────────────────
export type IncomeStatus = Schema['domain.IncomeStatus'] | '';
export type Household = Loose<
  DeepStrict<Schema['domain.Household']>,
  'annualHouseholdIncome' | 'childrenCountForFees' | 'parents' | 'children' | 'membershipParentId'
>;
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
export type Member = Loose<
  DeepStrict<Schema['domain.Member']>,
  | 'email'
  | 'phone'
  | 'street'
  | 'streetNo'
  | 'postalCode'
  | 'city'
  | 'householdId'
  | 'membershipEnd'
  | 'household'
>;
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
export type Parent = Loose<
  DeepStrict<Schema['domain.Parent']>,
  | 'householdId'
  | 'memberId'
  | 'birthDate'
  | 'email'
  | 'phone'
  | 'street'
  | 'streetNo'
  | 'postalCode'
  | 'city'
  | 'children'
  | 'household'
  | 'member'
>;
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

export type FeeExpectation = Loose<
  DeepStrict<Schema['domain.FeeExpectation']>,
  | 'householdId'
  | 'month'
  | 'paidAt'
  | 'matchedBy'
  | 'matchedAmount'
  | 'remaining'
  | 'partialMatches'
  | 'reminderForId'
  | 'reconciliationYear'
  | 'child'
>;
export type PaymentMatch = Loose<
  DeepStrict<Schema['domain.PaymentMatch']>,
  'confidence' | 'matchedBy' | 'transaction' | 'expectation'
>;
export type FeeOverview = DeepStrict<Schema['domain.FeeOverview']>;
export type MonthSummary = DeepStrict<Schema['domain.MonthSummary']>;
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
export type BankTransaction = Loose<
  DeepStrict<Schema['domain.BankTransaction']>,
  'payerName' | 'payerIban' | 'description' | 'importBatchId' | 'matchedAmount' | 'matches'
>;
export type MatchSuggestion = Loose<
  DeepStrict<Schema['domain.MatchSuggestion']>,
  'expectation' | 'expectations' | 'child' | 'detectedType'
> & {
  transaction: BankTransaction;
};
export type ImportResult = Loose<
  DeepStrict<Schema['service.ImportResult']>,
  'warningList'
> & {
  suggestions: MatchSuggestion[];
};
export interface MatchConfirmation {
  transactionId: string;
  expectationId: string;
}
export type ConfirmResult = DeepStrict<Schema['ConfirmMatchResponse']>;
export type ImportBatch = Loose<DeepStrict<Schema['domain.ImportBatch']>, 'dateFrom' | 'dateTo'>;

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
export type KnownIBAN = Loose<
  DeepStrict<Schema['domain.KnownIBAN']>,
  | 'payerName'
  | 'childId'
  | 'reason'
  | 'originalTransactionId'
  | 'originalDescription'
  | 'originalAmount'
  | 'child'
>;
export type KnownIBANSummary = Loose<
  DeepStrict<Schema['ChildTrustedIBANsResponse']>,
  'payerName'
>;
export type RescanResult = DeepStrict<Schema['RescanResponse']>;
export type DismissResult = DeepStrict<Schema['DismissTransactionResponse']>;
export type HideResult = DeepStrict<Schema['HideTransactionResponse']>;
export type UnmatchResult = DeepStrict<Schema['UnmatchTransactionResponse']>;
export type ChildUnmatchedSuggestionsResponse = Loose<
  DeepStrict<Schema['ChildUnmatchedSuggestionsResponse']>,
  'suggestions'
> & {
  suggestions: MatchSuggestion[];
};
export interface AllocationInput {
  expectationId: string;
  amount: number;
}
export type AllocateTransactionResult = DeepStrict<Schema['AllocateTransactionResponse']>;

// ── Transaction Warnings ─────────────────────────────────────────────────────
export type WarningType = Schema['domain.WarningType'];
export type ResolutionType = Schema['domain.ResolutionType'];
export type TransactionWarning = Loose<
  DeepStrict<Schema['domain.TransactionWarning']>,
  | 'expectedAmount'
  | 'actualAmount'
  | 'childId'
  | 'matchedFeeId'
  | 'resolvedAt'
  | 'resolvedBy'
  | 'resolutionType'
  | 'resolutionNote'
  | 'transaction'
  | 'child'
  | 'matchedFee'
> & {
  expectedAmount?: number | null;
  actualAmount?: number | null;
};
export type ResolveLateFeeResult = DeepStrict<Schema['ResolveLateFeeResponse']>;

// ── Child Import ─────────────────────────────────────────────────────────────
export type ChildImportParseResult = DeepStrict<Schema['service.ChildImportParseResult']>;
export interface ChildImportPreviewRequest {
  fileContent: string; // Base64 encoded CSV content
  separator: string;
  mapping: Record<string, number>; // systemField -> csvColumnIndex
  skipHeader: boolean;
}
// NOTE: the preview/row family below is kept as plain interfaces instead of
// derived DeepStrict+intersection types: vue-tsc resolves that combination
// differently inside SFCs and wrongly requires the re-added optional fields.
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

export type ChildcareFeeResult = Loose<
  DeepStrict<Schema['domain.ChildcareFeeResult']>,
  'notes'
> & {
  notes: string[];
};

// ── Ledger ───────────────────────────────────────────────────────────────────
export type LedgerEntry = Loose<
  DeepStrict<Schema['LedgerEntry']>,
  'feeType' | 'year' | 'month' | 'isPaid' | 'paidAt'
>;
export type LedgerSummary = DeepStrict<Schema['LedgerSummary']>;
export type ChildLedger = Loose<DeepStrict<Schema['ChildLedger']>, 'child' | 'entries'> & {
  entries: LedgerEntry[];
};

// ── Stichtagsmeldung ─────────────────────────────────────────────────────────
export type StichtagsmeldungStats = DeepStrict<Schema['StichtagsmeldungStats']>;
export type StichtagsmeldungReport = DeepStrict<Schema['StichtagsmeldungReport']>;
export type U3IncomeBreakdown = DeepStrict<Schema['U3IncomeBreakdown']>;
export type CareHoursBreakdownItem = Omit<
  DeepStrict<Schema['CareHoursBreakdownItem']>,
  'careHours'
> & {
  careHours?: number | null;
};
export type LegalHoursBreakdownItem = Omit<
  DeepStrict<Schema['LegalHoursBreakdownItem']>,
  'legalHours'
> & {
  legalHours?: number | null;
};
export type U3ChildDetail = Loose<
  DeepStrict<Schema['handler.U3ChildDetailResponse']>,
  'householdIncome' | 'incomeStatus'
> & {
  householdIncome: number | null;
  incomeStatus: string | null;
};

// ── Fee Coverage Timeline ────────────────────────────────────────────────────
export type CoverageStatus = NonNullable<Schema['handler.FeeCoverageResponse']['status']>;
export type CoveredTransaction = Loose<
  DeepStrict<Schema['handler.CoveredTransactionResponse']>,
  'description'
>;
export type FeeCoverage = Loose<
  DeepStrict<Schema['handler.FeeCoverageResponse']>,
  'transactions'
> & {
  transactions: CoveredTransaction[];
};

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
export type EinstufungMonthRow = DeepStrict<Schema['domain.EinstufungMonthRow']>;
export type Einstufung = Loose<
  DeepStrict<Schema['Einstufung']>,
  | 'validUntil'
  | 'sourceEinstufungId'
  | 'changeDate'
  | 'notes'
  | 'monthlyTable'
  | 'child'
  | 'household'
> & {
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
export type CreditReviewPeriod = DeepStrict<Schema['CreditReviewPeriod']>;
export type ChildcareExpectationSyncResult = DeepStrict<Schema['ChildcareExpectationSyncResult']>;
export type CreateFollowUpEinstufungResponse = Loose<
  DeepStrict<Schema['CreateFollowUpEinstufungResponse']>,
  'einstufung' | 'expectationChanges'
> & {
  einstufung: Einstufung;
  expectationChanges: ChildcareExpectationSyncResult;
};
export type CalculateIncomeResponse = DeepStrict<Schema['CalculateIncomeResponse']>;
