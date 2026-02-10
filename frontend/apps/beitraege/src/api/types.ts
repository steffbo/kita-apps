// API Types for the fees backend

// Auth
export interface LoginRequest {
  email: string;
  password: string;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
}

export interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  role: 'ADMIN' | 'USER';
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// Reminder settings and runs
export interface ReminderSettingsResponse {
  autoEnabled: boolean;
}

export interface UpdateReminderSettingsRequest {
  autoEnabled: boolean;
}

export type ReminderRunStage = 'auto' | 'initial' | 'final' | 'none';

export interface ReminderRunResponse {
  stage: ReminderRunStage;
  date: string;
  recipient: string;
  unpaidCount: number;
  reminderCreated: number;
  emailSent: boolean;
  dryRun: boolean;
  message?: string;
}

export type EmailLogType = 'REMINDER_INITIAL' | 'REMINDER_FINAL' | 'PASSWORD_RESET' | string;

export interface EmailLog {
  id: string;
  sentAt: string;
  toEmail: string;
  subject: string;
  body?: string | null;
  emailType: EmailLogType;
  sentBy?: string | null;
}

// Children
export interface Child {
  id: string;
  householdId?: string;
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
  legalHours?: number;
  legalHoursUntil?: string;
  careHours?: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  parents?: Parent[];
  household?: Household;
}

export interface NextMemberNumberResponse {
  memberNumber: string;
}

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
  legalHours?: number;
  legalHoursUntil?: string;
  careHours?: number;
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
  legalHours?: number;
  legalHoursUntil?: string;
  careHours?: number;
  isActive?: boolean;
  householdId?: string;
}

// Households
export type IncomeStatus = '' | 'PROVIDED' | 'MAX_ACCEPTED' | 'PENDING' | 'NOT_REQUIRED' | 'HISTORIC' | 'FOSTER_FAMILY';

export interface Household {
  id: string;
  name: string;
  annualHouseholdIncome?: number;
  incomeStatus: IncomeStatus;
  childrenCountForFees?: number;
  createdAt: string;
  updatedAt: string;
  parents?: Parent[];
  children?: Child[];
}

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

// Members (Vereinsmitglieder - can exist independently of children)
export interface Member {
  id: string;
  memberNumber: string;
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
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  household?: Household;
}

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

// Parents

export interface Parent {
  id: string;
  householdId?: string;
  memberId?: string; // Reference to Member if parent is also a Vereinsmitglied
  firstName: string;
  lastName: string;
  birthDate?: string;
  email?: string;
  phone?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  createdAt: string;
  updatedAt: string;
  children?: Child[];
  household?: Household;
  member?: Member; // Linked member if parent is a Vereinsmitglied
}

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

// Fees
export type FeeType = 'MEMBERSHIP' | 'FOOD' | 'CHILDCARE' | 'REMINDER';
export type FeeStatus = 'OPEN' | 'PAID' | 'OVERDUE';

export interface FeeExpectation {
  id: string;
  childId: string;
  feeType: FeeType;
  year: number;
  month?: number;
  amount: number;
  dueDate: string;
  createdAt: string;
  child?: Child;
  isPaid: boolean;
  paidAt?: string;
  matchedBy?: PaymentMatch;
  matchedAmount?: number;
  remaining?: number;
  partialMatches?: PaymentMatch[];
  reminderForId?: string; // Links REMINDER fee to original fee
}

export interface PaymentMatch {
  id: string;
  transactionId: string;
  expectationId: string;
  amount: number;
  matchType: 'AUTO' | 'MANUAL';
  confidence?: number;
  matchedAt: string;
  matchedBy?: string;
  transaction?: BankTransaction;
}

export interface FeeOverview {
  totalOpen: number;
  totalPaid: number;
  totalOverdue: number;
  amountOpen: number;
  amountPaid: number;
  amountOverdue: number;
  byMonth: MonthSummary[];
  childrenWithOpenFees: number;
  openMembershipCount: number;
  openFoodCount: number;
  openChildcareCount: number;
}

export interface MonthSummary {
  year: number;
  month: number;
  openCount: number;
  paidCount: number;
  openAmount: number;
  paidAmount: number;
}

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

// Payment Match
export interface PaymentMatch {
  id: string;
  transactionId: string;
  expectationId: string;
  amount: number;
  matchType: 'AUTO' | 'MANUAL';
  confidence?: number;
  matchedAt: string;
  matchedBy?: string;
  expectation?: FeeExpectation;
}

// Bank Transactions
export interface BankTransaction {
  id: string;
  bookingDate: string;
  valueDate: string;
  payerName?: string;
  payerIban?: string;
  description?: string;
  amount: number;
  currency: string;
  importBatchId?: string;
  importedAt: string;
  matches?: PaymentMatch[];
}

export interface MatchSuggestion {
  transaction: BankTransaction;
  expectation?: FeeExpectation;
  expectations?: FeeExpectation[];
  child?: Child;
  detectedType?: FeeType;
  confidence: number;
  matchedBy: string;
}

export interface ImportResult {
  batchId: string;
  fileName: string;
  totalRows: number;
  imported: number;
  skipped: number;
  suggestions: MatchSuggestion[];
}

export interface MatchConfirmation {
  transactionId: string;
  expectationId: string;
}

export interface ConfirmResult {
  confirmed: number;
  failed: number;
}

export interface ImportBatch {
  id: string;
  fileName: string;
  transactionCount: number;
  matchedCount: number;
  importedAt: string;
  importedBy: string;
  importedByEmail?: string;
  dateFrom?: string;
  dateTo?: string;
}

export type BankingSyncStatusType = 'idle' | 'running' | 'waiting_for_2fa' | 'success' | 'error';

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

// Paginated responses
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  offset: number;
  limit: number;
}

// API Error
export interface ApiError {
  error: string;
  message?: string;
}

// Known IBANs (IBAN Learning System)
export type KnownIBANStatus = 'trusted' | 'blacklisted';

export interface KnownIBAN {
  iban: string;
  payerName?: string;
  status: KnownIBANStatus;
  childId?: string;
  reason?: string;
  originalTransactionId?: string;
  originalDescription?: string;
  originalAmount?: number;
  createdAt: string;
  updatedAt: string;
  child?: Child;
}

export interface KnownIBANSummary {
  iban: string;
  payerName?: string;
  transactionCount: number;
}

export interface RescanResult {
  scanned: number;
  autoMatched: number;
  newMatches: number;
  suggestions: MatchSuggestion[];
}

export interface DismissResult {
  iban: string;
  transactionsRemoved: number;
}

export interface HideResult {
  transactionId: string;
}

export interface UnmatchResult {
  transactionId: string;
  matchesRemoved: number;
  transactionDeleted: boolean;
}

export interface ChildUnmatchedSuggestionsResponse {
  childId: string;
  scanned: number;
  suggestions: MatchSuggestion[];
}

export interface AllocationInput {
  expectationId: string;
  amount: number;
}

export interface AllocateTransactionResult {
  transactionId: string;
  allocationsCreated: number;
  totalAllocated: number;
  overpayment: number;
}

// Transaction Warnings
export type WarningType =
  | 'AMOUNT_MISMATCH'
  | 'DUPLICATE_PAYMENT'
  | 'UNKNOWN_IBAN'
  | 'LATE_PAYMENT'
  | 'OVERPAYMENT';
export type ResolutionType = 'DISMISSED' | 'MATCHED' | 'AUTO_RESOLVED';

export interface TransactionWarning {
  id: string;
  warningType: WarningType;
  message: string;
  transactionId?: string;
  childId?: string;
  feeId?: string;
  matchedFeeId?: string; // For LATE_PAYMENT: the fee that was paid late
  createdAt: string;
  resolvedAt?: string;
  resolvedBy?: string;
  resolutionType?: ResolutionType;
  resolutionNote?: string;
  transaction?: BankTransaction;
  child?: Child;
  fee?: FeeExpectation;
  matchedFee?: FeeExpectation;
}

export interface ResolveLateFeeResult {
  warningId: string;
  lateFeeId: string;
  lateFeeAmount: number;
}

// Child Import Types
export interface ChildImportParseResult {
  headers: string[];
  sampleRows: string[][];
  detectedSeparator: string;
  totalRows: number;
}

export interface ChildImportPreviewRequest {
  fileContent: string; // Base64 encoded CSV content
  separator: string;
  mapping: Record<string, number>; // systemField -> csvColumnIndex
  skipHeader: boolean;
}

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
  legalHours?: number;
  careHours?: number;
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
  imported: number;
  errors: string[];
}

// System fields for mapping UI
export interface SystemField {
  key: string;
  label: string;
  required: boolean;
  group: 'child' | 'parent1' | 'parent2';
}

// Childcare Fee Calculation
export type ChildAgeType = 'krippe' | 'kindergarten';

export interface ChildcareFeeInput {
  childAgeType?: ChildAgeType;
  income: number;
  siblingsCount?: number;
  careHours?: number;
  highestRate?: boolean;
  fosterFamily?: boolean;
}

export interface ChildcareFeeResult {
  fee: number;
  baseFee: number;
  rule: string;
  discountFactor: number;
  discountPercent: number;
  showEntlastung: boolean;
  notes: string[];
}

// Ledger Types
export interface LedgerEntry {
  id: string;
  date: string;
  type: 'fee' | 'payment';
  description: string;
  feeType?: FeeType;
  year?: number;
  month?: number;
  debit: number;
  credit: number;
  balance: number;
  isPaid?: boolean;
  paidAt?: string;
  fee?: FeeExpectation;
  transaction?: BankTransaction;
}

export interface LedgerSummary {
  totalFees: number;
  totalPaid: number;
  totalOpen: number;
  openFeesCount: number;
  paidFeesCount: number;
  totalFeesCount: number;
}

export interface ChildLedger {
  childId: string;
  child?: Child;
  entries: LedgerEntry[];
  summary: LedgerSummary;
}

// Stichtagsmeldung
export interface StichtagsmeldungStats {
  nextStichtag: string;
  daysUntilStichtag: number;
  u3IncomeBreakdown: U3IncomeBreakdown;
  totalChildrenInKita: number;
}

export interface U3IncomeBreakdown {
  upTo20k: number;
  from20To35k: number;
  from35To55k: number;
  maxAccepted: number;
  fosterFamily: number;
  total: number;
}

export interface U3ChildDetail {
  id: string;
  memberNumber: string;
  firstName: string;
  lastName: string;
  birthDate: string;
  householdIncome: number | null;
  incomeStatus: string | null;
  isFosterFamily: boolean;
}

// Fee Coverage Timeline
export type CoverageStatus = 'UNPAID' | 'PARTIAL' | 'COVERED' | 'OVERPAID';

export interface CoveredTransaction {
  transactionId: string;
  amount: number;
  bookingDate: string;
  description?: string;
  isForThisMonth: boolean;
}

export interface FeeCoverage {
  year: number;
  month: number;
  expectedTotal: number;
  receivedTotal: number;
  balance: number;
  status: CoverageStatus;
  transactions: CoveredTransaction[];
}

// Einstufung (Fee Classification)

export interface IncomeDetails {
  grossIncome: number;
  otherIncome: number;
  socialSecurityShare: number;
  privateInsurance: number;
  tax: number;
  advertisingCosts: number;
  profit: number;
  welfareExpense: number;
  selfEmployedTax: number;
  parentalBenefit: number;
  maternityBenefit: number;
  insurances: number;
  maintenanceToPay: number;
  maintenanceReceived: number;
}

export interface HouseholdIncomeCalculation {
  parent1: IncomeDetails;
  parent2: IncomeDetails;
}

export interface EinstufungMonthRow {
  month: number;
  year: number;
  careHoursPerWeek: number;
  careType: string;
  childcareFee: number;
  foodFee: number;
  membershipFee: number;
}

export interface Einstufung {
  id: string;
  childId: string;
  householdId: string;
  year: number;
  validFrom: string;
  incomeCalculation: HouseholdIncomeCalculation;
  annualNetIncome: number;
  highestRateVoluntary: boolean;
  careHoursPerWeek: number;
  careType: ChildAgeType;
  childrenCount: number;
  monthlyChildcareFee: number;
  monthlyFoodFee: number;
  annualMembershipFee: number;
  feeRule: string;
  discountPercent: number;
  discountFactor: number;
  baseFee: number;
  notes?: string;
  monthlyTable?: EinstufungMonthRow[];
  createdAt: string;
  updatedAt: string;
  child?: Child;
  household?: Household;
}

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

export interface CalculateIncomeResponse {
  parent1NetIncome: number;
  parent2NetIncome: number;
  parent1FeeRelevantIncome: number;
  parent2FeeRelevantIncome: number;
  householdFeeIncome: number;
  householdFullIncome: number;
}
