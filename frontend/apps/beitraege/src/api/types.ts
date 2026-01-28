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
  reminderForId?: string; // Links REMINDER fee to original fee
}

export interface PaymentMatch {
  id: string;
  transactionId: string;
  expectationId: string;
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
}

export interface MatchSuggestion {
  transaction: BankTransaction;
  expectation?: FeeExpectation;
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

export interface RescanResult {
  scanned: number;
  suggestions: MatchSuggestion[];
}

export interface DismissResult {
  iban: string;
  transactionsRemoved: number;
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
  childrenCreated: number;
  childrenUpdated: number;
  parentsCreated: number;
  parentsLinked: number;
  errors: ChildImportError[];
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

