import type {
  LoginRequest,
  TokenPair,
  User,
  Child,
  CreateChildRequest,
  UpdateChildRequest,
  Parent,
  CreateParentRequest,
  UpdateParentRequest,
  FeeExpectation,
  FeeOverview,
  GenerateFeeRequest,
  GenerateFeeResult,
  ImportResult,
  MatchConfirmation,
  ConfirmResult,
  ImportBatch,
  BankTransaction,
  PaginatedResponse,
} from './types';

const API_BASE = '/api/fees/v1';

class ApiClient {
  private accessToken: string | null = null;

  setAccessToken(token: string | null) {
    this.accessToken = token;
  }

  private async request<T>(
    path: string,
    options: RequestInit = {}
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
      if (response.status === 401) {
        // Token expired or invalid
        this.accessToken = null;
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
    search?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<Child>> {
    const query = new URLSearchParams();
    if (params?.activeOnly) query.set('active', 'true');
    if (params?.search) query.set('search', params.search);
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.limit) query.set('limit', String(params.limit));
    const queryString = query.toString();
    const response = await this.request<PaginatedResponse<Child>>(`/children${queryString ? `?${queryString}` : ''}`);
    return this.normalizePaginated(response);
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

  // Parents endpoints
  async getParents(params?: {
    search?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<Parent>> {
    const query = new URLSearchParams();
    if (params?.search) query.set('search', params.search);
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

  // Fees endpoints
  async getFees(params?: {
    year?: number;
    month?: number;
    feeType?: string;
    childId?: string;
    offset?: number;
    limit?: number;
  }): Promise<PaginatedResponse<FeeExpectation>> {
    const query = new URLSearchParams();
    if (params?.year) query.set('year', String(params.year));
    if (params?.month) query.set('month', String(params.month));
    if (params?.feeType) query.set('feeType', params.feeType);
    if (params?.childId) query.set('childId', params.childId);
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

  async updateFee(id: string, amount: number): Promise<FeeExpectation> {
    return this.request<FeeExpectation>(`/fees/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ amount }),
    });
  }

  async deleteFee(id: string): Promise<void> {
    return this.request<void>(`/fees/${id}`, { method: 'DELETE' });
  }

  async calculateChildcareFee(income: number): Promise<{ amount: number; bracket: string }> {
    return this.request<{ amount: number; bracket: string }>(`/childcare-fee/calculate?income=${income}`);
  }

  // Import endpoints
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

  async getUnmatchedTransactions(offset?: number, limit?: number): Promise<PaginatedResponse<BankTransaction>> {
    const query = new URLSearchParams();
    if (offset) query.set('offset', String(offset));
    if (limit) query.set('limit', String(limit));
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
}

export const api = new ApiClient();
export default api;
