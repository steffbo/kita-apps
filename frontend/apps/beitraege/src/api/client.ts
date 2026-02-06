import type {
  LoginRequest,
  TokenPair,
  User,
  Child,
  CreateChildRequest,
  UpdateChildRequest,
  NextMemberNumberResponse,
  Parent,
  CreateParentRequest,
  UpdateParentRequest,
  Household,
  CreateHouseholdRequest,
  UpdateHouseholdRequest,
  Member,
  CreateMemberRequest,
  UpdateMemberRequest,
  FeeExpectation,
  FeeOverview,
  GenerateFeeRequest,
  GenerateFeeResult,
  CreateFeeRequest,
  ReminderSettingsResponse,
  UpdateReminderSettingsRequest,
  ReminderRunResponse,
  EmailLog,
  ChildLedger,
  FeeCoverage,
  ImportResult,
  MatchConfirmation,
  ConfirmResult,
  ImportBatch,
  BankTransaction,
  PaginatedResponse,
  KnownIBAN,
  KnownIBANSummary,
  RescanResult,
  DismissResult,
  HideResult,
  UnmatchResult,
  ChildUnmatchedSuggestionsResponse,
  AllocationInput,
  AllocateTransactionResult,
  TransactionWarning,
  ResolveLateFeeResult,
  ChildImportParseResult,
  ChildImportPreviewRequest,
  ChildImportPreviewResult,
  ChildImportExecuteRequest,
  ChildImportExecuteResult,
  ChildcareFeeResult,
  MatchSuggestion,
  BankingSyncStatus,
} from './types';

const API_BASE = '/api/fees/v1';

class ApiClient {
  private accessToken: string | null = null;
  private refreshToken: string | null = null;
  private refreshPromise: Promise<boolean> | null = null;
  private onTokenRefreshed: ((tokens: { accessToken: string; refreshToken: string }) => void) | null = null;
  private onAuthFailed: (() => void) | null = null;

  setAccessToken(token: string | null) {
    this.accessToken = token;
  }

  setRefreshToken(token: string | null) {
    this.refreshToken = token;
  }

  // Callback when tokens are refreshed - auth store should use this to update its state
  setOnTokenRefreshed(callback: (tokens: { accessToken: string; refreshToken: string }) => void) {
    this.onTokenRefreshed = callback;
  }

  // Callback when auth completely fails (refresh failed) - auth store should clear state
  setOnAuthFailed(callback: () => void) {
    this.onAuthFailed = callback;
  }

  private async tryRefreshToken(): Promise<boolean> {
    // If already refreshing, wait for that to complete
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    if (!this.refreshToken) {
      return false;
    }

    this.refreshPromise = (async () => {
      try {
        const response = await fetch(`${API_BASE}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refreshToken: this.refreshToken }),
        });

        if (!response.ok) {
          return false;
        }

        const tokens = await response.json();
        this.accessToken = tokens.accessToken;
        this.refreshToken = tokens.refreshToken;

        // Notify auth store to update its state
        if (this.onTokenRefreshed) {
          this.onTokenRefreshed(tokens);
        }

        return true;
      } catch {
        return false;
      } finally {
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise;
  }

  private async request<T>(
    path: string,
    options: RequestInit = {},
    isRetry = false
  ): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.accessToken) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.accessToken}`;
    }

    const response = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      if (response.status === 401 && !isRetry) {
        // Token expired - try to refresh
        const refreshed = await this.tryRefreshToken();
        if (refreshed) {
          // Retry the original request with new token
          return this.request<T>(path, options, true);
        }
        // Refresh failed - notify auth store and throw
        if (this.onAuthFailed) {
          this.onAuthFailed();
        }
        throw new Error('Unauthorized');
      }
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(error.error || error.message || `HTTP ${response.status}`);
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  // Auth endpoints
  async login(credentials: LoginRequest): Promise<TokenPair> {
    return this.request<TokenPair>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }

  async refresh(refreshToken: string): Promise<TokenPair> {
    return this.request<TokenPair>('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refreshToken }),
    });
  }

  async logout(): Promise<void> {
    return this.request<void>('/auth/logout', { method: 'POST' });
  }

  async me(): Promise<User> {
    return this.request<User>('/auth/me');
  }

  // Helper to normalize paginated responses (Go returns null for empty slices)
  private normalizePaginated<T>(response: PaginatedResponse<T>): PaginatedResponse<T> {
    return {
      ...response,
      data: response.data ?? [],
    };
  }

  // Children endpoints
  async getChildren(params?: {
    activeOnly?: boolean;
    u3Only?: boolean;
    hasWarnings?: boolean;
    hasOpenFees?: boolean;
    search?: string;
    sortBy?: string;
    sortDir?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<Child>> {
    const query = new URLSearchParams();
    if (params?.activeOnly) query.set('active', 'true');
    if (params?.u3Only) query.set('u3Only', 'true');
    if (params?.hasWarnings) query.set('hasWarnings', 'true');
    if (params?.hasOpenFees) query.set('hasOpenFees', 'true');
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<Child>>(`/children${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getNextChildMemberNumber(): Promise<NextMemberNumberResponse> {
    return this.request<NextMemberNumberResponse>('/children/next-member-number');
  }

  async getChild(id: string): Promise<Child> {
    return this.request<Child>(`/children/${id}`);
  }

  async createChild(data: CreateChildRequest): Promise<Child> {
    return this.request<Child>('/children', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateChild(id: string, data: UpdateChildRequest): Promise<Child> {
    return this.request<Child>(`/children/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteChild(id: string): Promise<void> {
    return this.request<void>(`/children/${id}`, { method: 'DELETE' });
  }

  async linkParent(childId: string, parentId: string, isPrimary: boolean): Promise<void> {
    return this.request<void>(`/children/${childId}/parents`, {
      method: 'POST',
      body: JSON.stringify({ parentId, isPrimary }),
    });
  }

  async unlinkParent(childId: string, parentId: string): Promise<void> {
    return this.request<void>(`/children/${childId}/parents/${parentId}`, {
      method: 'DELETE',
    });
  }

  async getChildLedger(childId: string, year?: number): Promise<ChildLedger> {
    const query = year ? `?year=${year}` : '';
    const response = await this.request<ChildLedger>(`/children/${childId}/ledger${query}`);
    return {
      ...response,
      entries: response.entries ?? [],
    };
  }

  async getChildTimeline(childId: string, year?: number): Promise<FeeCoverage[]> {
    const query = year ? `?year=${year}` : '';
    return this.request<FeeCoverage[]>(`/children/${childId}/timeline${query}`);
  }

  // Parents endpoints
  async getParents(params?: {
    search?: string;
    sortBy?: string;
    sortDir?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<Parent>> {
    const query = new URLSearchParams();
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<Parent>>(`/parents${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getParent(id: string): Promise<Parent> {
    return this.request<Parent>(`/parents/${id}`);
  }

  async createParent(data: CreateParentRequest): Promise<Parent> {
    return this.request<Parent>('/parents', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateParent(id: string, data: UpdateParentRequest): Promise<Parent> {
    return this.request<Parent>(`/parents/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteParent(id: string): Promise<void> {
    return this.request<void>(`/parents/${id}`, { method: 'DELETE' });
  }

  // Create a member from parent data and link them
  async createMemberFromParent(parentId: string, membershipStart?: string): Promise<Parent> {
    return this.request<Parent>(`/parents/${parentId}/member`, {
      method: 'POST',
      body: JSON.stringify({ membershipStart: membershipStart || new Date().toISOString().split('T')[0] }),
    });
  }

  // Unlink a member from a parent (does not delete the member)
  async unlinkMemberFromParent(parentId: string): Promise<Parent> {
    return this.request<Parent>(`/parents/${parentId}/member`, {
      method: 'DELETE',
    });
  }

  // Households endpoints
  async getHouseholds(params?: {
    search?: string;
    sortBy?: string;
    sortDir?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<Household>> {
    const query = new URLSearchParams();
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<Household>>(`/households${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getHousehold(id: string): Promise<Household> {
    return this.request<Household>(`/households/${id}`);
  }

  async createHousehold(data: CreateHouseholdRequest): Promise<Household> {
    return this.request<Household>('/households', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateHousehold(id: string, data: UpdateHouseholdRequest): Promise<Household> {
    return this.request<Household>(`/households/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteHousehold(id: string): Promise<void> {
    return this.request<void>(`/households/${id}`, { method: 'DELETE' });
  }

  async linkParentToHousehold(householdId: string, parentId: string): Promise<void> {
    return this.request<void>(`/households/${householdId}/parents`, {
      method: 'POST',
      body: JSON.stringify({ parentId }),
    });
  }

  async linkChildToHousehold(householdId: string, childId: string): Promise<void> {
    return this.request<void>(`/households/${householdId}/children`, {
      method: 'POST',
      body: JSON.stringify({ childId }),
    });
  }

  // Members (Vereinsmitglieder) endpoints
  async getMembers(params?: {
    activeOnly?: boolean;
    search?: string;
    sortBy?: string;
    sortDir?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<Member>> {
    const query = new URLSearchParams();
    if (params?.activeOnly) query.set('active', 'true');
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<Member>>(`/members${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getMember(id: string): Promise<Member> {
    return this.request<Member>(`/members/${id}`);
  }

  async createMember(data: CreateMemberRequest): Promise<Member> {
    return this.request<Member>('/members', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateMember(id: string, data: UpdateMemberRequest): Promise<Member> {
    return this.request<Member>(`/members/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteMember(id: string): Promise<void> {
    return this.request<void>(`/members/${id}`, { method: 'DELETE' });
  }

  // Fees endpoints
  async getFees(params?: {
    year?: number;
    month?: number;
    feeType?: string;
    status?: string;
    childId?: string;
    search?: string;
    sortBy?: string;
    sortDir?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<FeeExpectation>> {
    const query = new URLSearchParams();
    if (params?.year) query.set('year', String(params.year));
    if (params?.month) query.set('month', String(params.month));
    if (params?.feeType) query.set('type', params.feeType);
    if (params?.status) query.set('status', params.status);
    if (params?.childId) query.set('childId', params.childId);
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<FeeExpectation>>(`/fees${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getFee(id: string): Promise<FeeExpectation> {
    return this.request<FeeExpectation>(`/fees/${id}`);
  }

  async getFeeOverview(year?: number): Promise<FeeOverview> {
    const query = year ? `?year=${year}` : '';
    const response = await this.request<FeeOverview>(`/fees/overview${query}`);
    return {
      ...response,
      byMonth: response.byMonth ?? [],
    };
  }

  async generateFees(data: GenerateFeeRequest): Promise<GenerateFeeResult> {
    return this.request<GenerateFeeResult>('/fees/generate', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async createFee(data: CreateFeeRequest): Promise<FeeExpectation> {
    return this.request<FeeExpectation>('/fees', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateFee(id: string, amount: number): Promise<FeeExpectation> {
    return this.request<FeeExpectation>(`/fees/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ amount }),
    });
  }

  async deleteFee(id: string): Promise<void> {
    return this.request<void>(`/fees/${id}`, { method: 'DELETE' });
  }

  async createReminder(feeId: string): Promise<FeeExpectation> {
    return this.request<FeeExpectation>(`/fees/${feeId}/reminder`, {
      method: 'POST',
    });
  }

  async getReminderSettings(): Promise<ReminderSettingsResponse> {
    return this.request<ReminderSettingsResponse>('/fees/reminders/settings');
  }

  async updateReminderSettings(data: UpdateReminderSettingsRequest): Promise<ReminderSettingsResponse> {
    return this.request<ReminderSettingsResponse>('/fees/reminders/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async runReminders(params?: {
    stage?: 'auto' | 'initial' | 'final';
    date?: string;
    dryRun?: boolean;
  }): Promise<ReminderRunResponse> {
    const query = new URLSearchParams();
    if (params?.stage) query.set('stage', params.stage);
    if (params?.date) query.set('date', params.date);
    if (typeof params?.dryRun === 'boolean') query.set('dryRun', params.dryRun ? 'true' : 'false');
    const queryString = query.toString();
    return this.request<ReminderRunResponse>(`/fees/reminders/run${queryString ? `?${queryString}` : ''}`, {
      method: 'POST',
    });
  }

  async getEmailLogs(params?: { offset?: number; limit?: number }): Promise<PaginatedResponse<EmailLog>> {
    const query = new URLSearchParams();
    if (typeof params?.offset === 'number') query.set('offset', String(params.offset));
    if (typeof params?.limit === 'number') query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<EmailLog>>(`/fees/email-logs${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async calculateChildcareFee(params: {
    income: number;
    childAgeType?: 'krippe' | 'kindergarten';
    siblingsCount?: number;
    careHours?: number;
    highestRate?: boolean;
    fosterFamily?: boolean;
  }): Promise<ChildcareFeeResult> {
    const searchParams = new URLSearchParams();
    searchParams.set('income', params.income.toString());
    if (params.childAgeType) searchParams.set('childAgeType', params.childAgeType);
    if (params.siblingsCount) searchParams.set('siblingsCount', params.siblingsCount.toString());
    if (params.careHours) searchParams.set('careHours', params.careHours.toString());
    if (params.highestRate) searchParams.set('highestRate', 'true');
    if (params.fosterFamily) searchParams.set('fosterFamily', 'true');
    return this.request<ChildcareFeeResult>(`/childcare-fee/calculate?${searchParams.toString()}`);
  }

  // Import endpoints
  async runBankingSync(): Promise<BankingSyncStatus> {
    return this.request<BankingSyncStatus>('/banking-sync/run', { method: 'POST' });
  }

  async getBankingSyncStatus(): Promise<BankingSyncStatus> {
    return this.request<BankingSyncStatus>('/banking-sync/status');
  }

  async uploadCSV(file: File): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);

    const headers: HeadersInit = {};
    if (this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`;
    }

    const response = await fetch(`${API_BASE}/import/upload`, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }));
      throw new Error(error.error || 'Upload failed');
    }

    const result: ImportResult = await response.json();
    return {
      ...result,
      suggestions: result.suggestions ?? [],
    };
  }

  async confirmMatches(matches: MatchConfirmation[]): Promise<ConfirmResult> {
    return this.request<ConfirmResult>('/import/confirm', {
      method: 'POST',
      body: JSON.stringify({ matches }),
    });
  }

  async getImportHistory(offset?: number, limit?: number): Promise<PaginatedResponse<ImportBatch>> {
    const query = new URLSearchParams();
    if (offset) query.set('offset', String(offset));
    if (limit) query.set('limit', String(limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<ImportBatch>>(`/import/history${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getUnmatchedTransactions(params?: {
    offset?: number;
    limit?: number;
    search?: string;
    sortBy?: string;
    sortDir?: string;
  }): Promise<PaginatedResponse<BankTransaction>> {
    const query = new URLSearchParams();
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<BankTransaction>>(`/import/transactions${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async createManualMatch(transactionId: string, expectationId: string): Promise<void> {
    return this.request<void>('/import/match', {
      method: 'POST',
      body: JSON.stringify({ transactionId, expectationId }),
    });
  }

  async getMatchedTransactions(params?: {
    offset?: number;
    limit?: number;
    search?: string;
    sortBy?: string;
    sortDir?: string;
  }): Promise<PaginatedResponse<BankTransaction>> {
    const query = new URLSearchParams();
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.search) query.set('search', params.search);
    if (params?.sortBy) query.set('sortBy', params.sortBy);
    if (params?.sortDir) query.set('sortDir', params.sortDir);
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<BankTransaction>>(`/import/transactions/matched${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getTransactionSuggestions(transactionId: string): Promise<MatchSuggestion | null> {
    return this.request<MatchSuggestion | null>(`/import/transactions/${transactionId}/suggestions`);
  }

  // IBAN Learning System endpoints
  async rescanTransactions(): Promise<RescanResult> {
    const result = await this.request<RescanResult>('/import/rescan', {
      method: 'POST',
    });
    return {
      ...result,
      suggestions: result.suggestions ?? [],
    };
  }

  async dismissTransaction(transactionId: string): Promise<DismissResult> {
    return this.request<DismissResult>(`/import/transactions/${transactionId}/dismiss`, {
      method: 'POST',
    });
  }

  async hideTransaction(transactionId: string): Promise<HideResult> {
    return this.request<HideResult>(`/import/transactions/${transactionId}/hide`, {
      method: 'POST',
    });
  }

  async unmatchTransaction(transactionId: string, options?: { deleteTransaction?: boolean }): Promise<UnmatchResult> {
    return this.request<UnmatchResult>(`/import/transactions/${transactionId}/unmatch`, {
      method: 'POST',
      body: JSON.stringify({ deleteTransaction: options?.deleteTransaction ?? false }),
    });
  }

  async allocateTransaction(transactionId: string, allocations: AllocationInput[]): Promise<AllocateTransactionResult> {
    return this.request<AllocateTransactionResult>(`/import/transactions/${transactionId}/allocate`, {
      method: 'POST',
      body: JSON.stringify({ allocations }),
    });
  }

  async getChildUnmatchedSuggestions(
    childId: string,
    params?: { minConfidence?: number; limit?: number }
  ): Promise<ChildUnmatchedSuggestionsResponse> {
    const query = new URLSearchParams();
    if (typeof params?.minConfidence === 'number') {
      query.set('minConfidence', String(params.minConfidence));
    }
    if (typeof params?.limit === 'number') {
      query.set('limit', String(params.limit));
    }
    const queryString = query.toString();
    return this.request<ChildUnmatchedSuggestionsResponse>(
      `/import/transactions/unmatched/child/${childId}${queryString ? `?${queryString}` : ''}`
    );
  }

  async getBlacklist(offset?: number, limit?: number): Promise<PaginatedResponse<KnownIBAN>> {
    const query = new URLSearchParams();
    if (offset) query.set('offset', String(offset));
    if (limit) query.set('limit', String(limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<KnownIBAN>>(`/import/blacklist${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async removeFromBlacklist(iban: string): Promise<void> {
    return this.request<void>(`/import/blacklist/${encodeURIComponent(iban)}`, {
      method: 'DELETE',
    });
  }

  async getTrustedIBANs(offset?: number, limit?: number): Promise<PaginatedResponse<KnownIBAN>> {
    const query = new URLSearchParams();
    if (offset) query.set('offset', String(offset));
    if (limit) query.set('limit', String(limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<KnownIBAN>>(`/import/trusted${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async getChildTrustedIBANs(childId: string): Promise<KnownIBANSummary[]> {
    return this.request<KnownIBANSummary[]>(`/import/trusted/child/${childId}`);
  }

  async linkIBANToChild(iban: string, childId: string): Promise<void> {
    return this.request<void>(`/import/trusted/${encodeURIComponent(iban)}/link`, {
      method: 'POST',
      body: JSON.stringify({ childId }),
    });
  }

  async unlinkIBANFromChild(iban: string): Promise<void> {
    return this.request<void>(`/import/trusted/${encodeURIComponent(iban)}/link`, {
      method: 'DELETE',
    });
  }

  // Warnings endpoints
  async getWarnings(offset?: number, limit?: number): Promise<PaginatedResponse<TransactionWarning>> {
    const query = new URLSearchParams();
    if (offset) query.set('offset', String(offset));
    if (limit) query.set('limit', String(limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<TransactionWarning>>(`/import/warnings${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
  }

  async dismissWarning(warningId: string, note?: string): Promise<void> {
    return this.request<void>(`/import/warnings/${warningId}/dismiss`, {
      method: 'POST',
      body: JSON.stringify({ note: note || '' }),
    });
  }

  async resolveLateFee(warningId: string): Promise<ResolveLateFeeResult> {
    return this.request<ResolveLateFeeResult>(`/import/warnings/${warningId}/resolve-late-fee`, {
      method: 'POST',
    });
  }

  // Child Import endpoints
  async parseChildImportCSV(file: File): Promise<ChildImportParseResult> {
    const formData = new FormData();
    formData.append('file', file);

    const headers: HeadersInit = {};
    if (this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`;
    }

    const response = await fetch(`${API_BASE}/children/import/parse`, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }));
      throw new Error(error.error || 'Upload failed');
    }

    const result: ChildImportParseResult = await response.json();
    return {
      ...result,
      sampleRows: result.sampleRows ?? [],
    };
  }

  async previewChildImport(request: ChildImportPreviewRequest): Promise<ChildImportPreviewResult> {
    const result = await this.request<ChildImportPreviewResult>('/children/import/preview', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return {
      ...result,
      rows: result.rows ?? [],
    };
  }

  async executeChildImport(request: ChildImportExecuteRequest): Promise<ChildImportExecuteResult> {
    const result = await this.request<ChildImportExecuteResult>('/children/import/execute', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return {
      ...result,
      errors: result.errors ?? [],
    };
  }

}

export const api = new ApiClient();
export default api;
