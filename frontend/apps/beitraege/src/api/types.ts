// API types for the fees backend.
//
// Response shapes are derived from the generated OpenAPI schema
// (src/api/schema.d.ts, produced by `bun run generate:api` from
// openapi/fees/openapi3.yaml) so field names/types can no longer drift
// from the backend contract.
//
// swag runs with --requiredByDefault: every JSON field is required unless the
// Go struct tag says `binding:"optional"`, which backend-fees sets on all
// `omitempty` and pointer fields. Generated optionality therefore mirrors
// what the API sends; requests and responses use schema aliases where possible.
import type { components } from './schema';

type Schema = components['schemas'];

// ── Elternstunden ────────────────────────────────────────────────────────────
export type ParentWorkOverview = Schema['service.ParentWorkOverview'];
export type ParentWorkAccount = Schema['domain.ParentWorkAccount'];
export type ParentWorkDetail = Schema['service.ParentWorkDetail'];
export type ParentWorkHouseholdOption = Schema['service.ParentWorkHouseholdOption'];
export type ParentWorkEntry = Schema['domain.ParentWorkEntry'];
export type ParentWorkEntryRequest = Omit<Schema['handler.parentWorkEntryRequest'], 'status'>;
export type BoardTerm = Schema['domain.BoardTerm'];
export type BoardTermRequest = Schema['handler.boardTermRequest'];
export type ParentWorkRule = Schema['domain.ParentWorkRule'];
export type ParentWorkRuleRequest = Schema['handler.parentWorkRuleRequest'];
export type ParentWorkOverrideRequest = Schema['handler.parentWorkOverrideRequest'];
export interface ParentWorkImportParseResult { headers: string[]; rows: string[][] }
export interface ParentWorkImportPreviewRow {
  index: number; workDate?: string; durationMinutes?: number; occasion?: string;
  memberName?: string; childName?: string; householdId?: string; householdName?: string;
  matchedBy?: 'child' | 'member'; errors: string[]; duplicate: boolean;
}
export interface ParentWorkImportExecuteRow {
  householdId: string; workDate: string; durationMinutes: number; occasion: string;
  memberName?: string; childName?: string;
}
export interface ParentWorkImportExecuteResult { created: number }

// ── Auth ─────────────────────────────────────────────────────────────────────
export type LoginRequest = Schema['LoginRequest'];
// The refresh token is never visible to scripts: the backend keeps it in the
// httpOnly cookie `fees_refresh`. Responses only carry the access token.
export type LoginResponse = Schema['LoginResponse'];
export type RefreshResponse = Schema['RefreshResponse'];
export type ImpersonationResponse = Schema['ImpersonationResponse'];
export type User = Schema['User'];
export type ChangePasswordRequest = Schema['ChangePasswordRequest'];

// ── User management (admin) ──────────────────────────────────────────────────
export type UserAccount = Schema['UserAccount'];
export type InvitationCandidate = Schema['repository.InvitationCandidate'];
export type InviteParentsResponse = Schema['InviteParentsResponse'];
export type UserRole = UserAccount['role'];
export type UserAccountRequest = Schema['UserAccountRequest'];
export type CreateUserAccountRequest = Schema['CreateUserAccountRequest'];

export type OwnOverview = Schema['handler.ownOverview'];
export type OwnChild = Schema['handler.ownChild'];
export type OwnFees = Schema['handler.ownFees'];
export type OwnWork = Schema['handler.ownWork'];
export type OwnWorkEntry = Schema['handler.ownWorkEntry'];
export type OwnWorkRequest = Schema['handler.ownWorkRequest'];
export type ParentReport = Schema['repository.ParentReport'];
export type ParentActivity = Schema['repository.Activity'];
export type DataChange = Schema['repository.DataChange'];
export type OwnContactRequest = Schema['handler.ownContactRequest'];
export type ReportRequest = Schema['service.ReportInput'];

// ── Reminder settings ───────────────────────────────────────────────
export type ReminderPaymentSettings = Schema['handler.ReminderPaymentSettingsPayload'];
export type ReminderSettingsResponse = 
  Schema['ReminderSettingsResponse'] & {
    payment: ReminderPaymentSettings;
  }
;
export interface UpdateReminderSettingsRequest {
  payment?: ReminderPaymentSettings;
}

// ── Family reminder cases ────────────────────────────────────────────────────
export type ReminderCaseFee = Schema['service.ReminderCaseFee'];
export type ReminderCase = Schema['service.ReminderCase'];
export type ReminderCasesResult = Schema['service.ReminderCasesResult'];
export type ReminderCasePreview = Schema['service.ReminderCasePreview'];
export type ReminderCaseSendResult = Schema['service.ReminderCaseSendResult'];
export type ReminderCaseRequest = Schema['ReminderCaseRequest'];
interface ReminderCaseConflict {
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

export type EmailLogType = Schema['EmailLogResponse']['emailType'];
export type EmailLog = Schema['EmailLogResponse'];

// ── Children ─────────────────────────────────────────────────────────────────
export type Child = Schema['domain.Child'];
export type NextMemberNumberResponse = Schema['NextMemberNumberResponse'];
// The generated schema includes fields from another create-child DTO; this
// endpoint's handler request struct only accepts the fields listed here.
export type CreateChildRequest = Omit<
  Schema['CreateChildRequest'],
  'membershipParentId' | 'name'
>;
export type UpdateChildRequest = Schema['UpdateChildRequest'];
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
export interface UpdateHouseholdRequest {
  name?: string;
  annualHouseholdIncome?: number;
  incomeStatus?: IncomeStatus;
  childrenCountForFees?: number;
}

// ── Members (Vereinsmitglieder - can exist independently of children) ───────
export type Member = Schema['domain.Member'];
export type CreateMemberRequest = Schema['CreateMemberRequest'];
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
export type FeeExpectation = Schema['domain.FeeExpectation'];
export type PaymentMatch = Schema['domain.PaymentMatch'];
export type FeeOverview = Schema['domain.FeeOverview'];
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

export type BankingSyncStatusType = Schema['BankingSyncStatus']['status'];
export type BankingSyncStatus = Schema['BankingSyncStatus'];

// ── Pagination envelope (response.Paginated; not generic in the spec) ────────
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  perPage: number;
  totalPages: number;
}

// ── Known IBANs (IBAN Learning System) ───────────────────────────────────────
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
interface ParentMatch {
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
export type ChildcareFeeResult = Schema['domain.ChildcareFeeResult'];

// ── Stichtagsmeldung ─────────────────────────────────────────────────────────
export type StichtagsmeldungStats = Schema['StichtagsmeldungStats'];
export type MemberCountAsOf = Schema['MemberCountAsOf'];
export type StichtagsmeldungReport = Schema['StichtagsmeldungReport'];
export type U3ChildDetail = Omit<Schema['handler.U3ChildDetailResponse'], 'householdIncome' | 'incomeStatus'> & {
  householdIncome: number | null;
  incomeStatus: string | null;
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
