import createClient, { type Middleware } from 'openapi-fetch';
import type { paths, components } from './schema.d';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

// Token storage
let accessToken: string | null = null;
let refreshToken: string | null = null;
let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

// Callback for handling auth failures (set by useAuth)
let onAuthFailure: (() => void) | null = null;

// Auth middleware - adds Authorization header
const authMiddleware: Middleware = {
  onRequest: ({ request }) => {
    if (accessToken) {
      request.headers.set('Authorization', `Bearer ${accessToken}`);
    }
    return request;
  },
  onResponse: async ({ request, response }) => {
    // Handle 401 Unauthorized responses
    if (response.status === 401) {
      // Don't handle auth failure for auth endpoints - let the caller handle those
      const isAuthEndpoint = request.url.includes('/auth/login') || 
                             request.url.includes('/auth/refresh') ||
                             request.url.includes('/auth/password-reset');
      
      if (isAuthEndpoint) {
        // Return response as-is for auth endpoints
        return response;
      }
      
      // For non-auth endpoints, try to refresh the token
      if (refreshToken) {
        const refreshed = await tryRefreshToken();
        
        if (refreshed) {
          // Retry the original request with new token
          const newRequest = new Request(request.url, {
            method: request.method,
            headers: new Headers(request.headers),
            body: request.body,
          });
          newRequest.headers.set('Authorization', `Bearer ${accessToken}`);
          
          // Return new response
          return fetch(newRequest);
        }
      }
      
      // Token refresh failed or no refresh token - trigger auth failure
      if (onAuthFailure) {
        onAuthFailure();
      }
    }
    
    return response;
  },
};

// Try to refresh the access token
async function tryRefreshToken(): Promise<boolean> {
  // If already refreshing, wait for that to complete
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }
  
  isRefreshing = true;
  
  refreshPromise = (async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refreshToken }),
      });
      
      if (!response.ok) {
        return false;
      }
      
      const data = await response.json();
      
      if (data.accessToken && data.refreshToken) {
        accessToken = data.accessToken;
        refreshToken = data.refreshToken;
        
        // Persist the new tokens
        const AUTH_STORAGE_KEY = 'kita_auth';
        const stored = localStorage.getItem(AUTH_STORAGE_KEY);
        if (stored) {
          try {
            const parsed = JSON.parse(stored);
            parsed.accessToken = data.accessToken;
            parsed.refreshToken = data.refreshToken;
            localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(parsed));
          } catch (e) {
            // Ignore parse errors
          }
        }
        
        return true;
      }
      
      return false;
    } catch (e) {
      return false;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();
  
  return refreshPromise;
}

// Create the client
export const apiClient = createClient<paths>({
  baseUrl: API_BASE_URL,
});

// Register middleware once
apiClient.use(authMiddleware);

// Set auth token
export function setAuthToken(token: string | null) {
  accessToken = token;
}

// Get current token
export function getAuthToken(): string | null {
  return accessToken;
}

// Set refresh token
export function setRefreshToken(token: string | null) {
  refreshToken = token;
}

// Set auth failure callback
export function setAuthFailureCallback(callback: () => void) {
  onAuthFailure = callback;
}

// Type exports for convenience
export type Employee = components['schemas']['Employee'];
export type CreateEmployeeRequest = components['schemas']['CreateEmployeeRequest'];
export type UpdateEmployeeRequest = components['schemas']['UpdateEmployeeRequest'];

// Enum types (defined manually as swag generates inline enums)
export type EmployeeRole = 'ADMIN' | 'EMPLOYEE';
export type AssignmentType = 'PERMANENT' | 'SPRINGER';
export type ScheduleEntryType = 'WORK' | 'VACATION' | 'SICK' | 'SPECIAL_LEAVE' | 'TRAINING' | 'EVENT';
export type TimeEntryType = 'WORK' | 'VACATION' | 'SICK' | 'SPECIAL_LEAVE' | 'TRAINING' | 'EVENT';
export type SpecialDayType = 'HOLIDAY' | 'CLOSURE' | 'TEAM_DAY' | 'EVENT';

export type Group = components['schemas']['Group'];
export type GroupWithMembers = components['schemas']['GroupWithMembers'];
export type CreateGroupRequest = components['schemas']['CreateGroupRequest'];
export type GroupAssignment = components['schemas']['GroupAssignment'];
export type GroupAssignmentRequest = components['schemas']['GroupAssignmentRequest'];

export type ScheduleEntry = components['schemas']['ScheduleEntry'];
export type CreateScheduleEntryRequest = components['schemas']['CreateScheduleEntryRequest'];
export type UpdateScheduleEntryRequest = components['schemas']['UpdateScheduleEntryRequest'];
export type WeekSchedule = components['schemas']['WeekSchedule'];
export type DaySchedule = components['schemas']['DaySchedule'];

export type TimeEntry = components['schemas']['TimeEntry'];
export type CreateTimeEntryRequest = components['schemas']['CreateTimeEntryRequest'];
export type UpdateTimeEntryRequest = components['schemas']['UpdateTimeEntryRequest'];
export type ClockInRequest = components['schemas']['ClockInRequest'];
export type ClockOutRequest = components['schemas']['ClockOutRequest'];
export type TimeScheduleComparison = components['schemas']['TimeScheduleComparison'];
export type DayComparison = components['schemas']['DayComparison'];

export type SpecialDay = components['schemas']['SpecialDay'];
export type CreateSpecialDayRequest = components['schemas']['CreateSpecialDayRequest'];

export type OverviewStatistics = components['schemas']['OverviewStatistics'];
export type EmployeeStatistics = components['schemas']['EmployeeStatistics'];
export type EmployeeStatisticsSummary = components['schemas']['EmployeeStatisticsSummary'];
export type WeeklyStatistics = components['schemas']['WeeklyStatistics'];

export type AuthResponse = components['schemas']['AuthResponse'];
export type LoginRequest = components['schemas']['LoginRequest'];

// Generic response types
export type MessageResponse = { message: string };
export type ErrorResponse = { error: string; message?: string };

export type { paths, components, operations } from './schema.d';
