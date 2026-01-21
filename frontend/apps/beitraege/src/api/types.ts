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
  memberNumber: string;
  firstName: string;
  lastName: string;
  birthDate: string;
  entryDate: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  parents?: Parent[];
}

export interface CreateChildRequest {
  memberNumber: string;
  firstName: string;
  lastName: string;
  birthDate: string;
  entryDate: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
}

export interface UpdateChildRequest {
  firstName?: string;
  lastName?: string;
  birthDate?: string;
  entryDate?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  isActive?: boolean;
}

// Parents
export interface Parent {
  id: string;
  firstName: string;
  lastName: string;
  birthDate?: string;
  email?: string;
  phone?: string;
  street?: string;
  streetNo?: string;
  postalCode?: string;
  city?: string;
  annualHouseholdIncome?: number;
  createdAt: string;
  updatedAt: string;
  children?: Child[];
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
  annualHouseholdIncome?: number;
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
  annualHouseholdIncome?: number;
}

// Fees
export type FeeType = 'MEMBERSHIP' | 'FOOD' | 'CHILDCARE';
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
