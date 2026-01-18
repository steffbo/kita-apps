import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import { apiClient, setAuthToken } from '../api';

interface User {
  id: number;
  email: string;
  firstName: string;
  lastName: string;
  role: 'ADMIN' | 'EMPLOYEE';
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
}

const AUTH_STORAGE_KEY = 'kita_auth';

// Global auth state
const authState = ref<AuthState>({
  user: null,
  accessToken: null,
  refreshToken: null,
});

// Initialize from localStorage
function initAuth() {
  const stored = localStorage.getItem(AUTH_STORAGE_KEY);
  if (stored) {
    try {
      const parsed = JSON.parse(stored);
      authState.value = parsed;
      if (parsed.accessToken) {
        setAuthToken(parsed.accessToken);
      }
    } catch (e) {
      localStorage.removeItem(AUTH_STORAGE_KEY);
    }
  }
}

// Save to localStorage
function persistAuth() {
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(authState.value));
}

export function useAuth() {
  const router = useRouter();
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  const isAuthenticated = computed(() => !!authState.value.accessToken);
  const user = computed(() => authState.value.user);
  const isAdmin = computed(() => authState.value.user?.role === 'ADMIN');

  async function login(email: string, password: string) {
    isLoading.value = true;
    error.value = null;

    try {
      const { data, error: apiError } = await apiClient.POST('/auth/login', {
        body: { email, password },
      });

      if (apiError || !data) {
        throw new Error(apiError?.message || 'Login fehlgeschlagen');
      }

      authState.value = {
        user: data.user as User,
        accessToken: data.accessToken!,
        refreshToken: data.refreshToken!,
      };

      setAuthToken(data.accessToken!);
      persistAuth();

      return data;
    } catch (e: any) {
      error.value = e.message || 'Login fehlgeschlagen';
      throw e;
    } finally {
      isLoading.value = false;
    }
  }

  async function logout() {
    authState.value = {
      user: null,
      accessToken: null,
      refreshToken: null,
    };
    setAuthToken(null);
    localStorage.removeItem(AUTH_STORAGE_KEY);
    router.push('/login');
  }

  async function refreshAccessToken() {
    if (!authState.value.refreshToken) {
      await logout();
      return;
    }

    try {
      const { data, error: apiError } = await apiClient.POST('/auth/refresh', {
        body: { refreshToken: authState.value.refreshToken },
      });

      if (apiError || !data) {
        throw new Error('Token refresh failed');
      }

      authState.value.accessToken = data.accessToken!;
      authState.value.refreshToken = data.refreshToken!;
      setAuthToken(data.accessToken!);
      persistAuth();
    } catch (e) {
      await logout();
    }
  }

  async function requestPasswordReset(email: string) {
    isLoading.value = true;
    error.value = null;

    try {
      const { error: apiError } = await apiClient.POST('/auth/password-reset/request', {
        body: { email },
      });

      if (apiError) {
        throw new Error(apiError.message || 'Anfrage fehlgeschlagen');
      }
    } catch (e: any) {
      error.value = e.message;
      throw e;
    } finally {
      isLoading.value = false;
    }
  }

  async function confirmPasswordReset(token: string, newPassword: string) {
    isLoading.value = true;
    error.value = null;

    try {
      const { error: apiError } = await apiClient.POST('/auth/password-reset/confirm', {
        body: { token, newPassword },
      });

      if (apiError) {
        throw new Error(apiError.message || 'Passwort konnte nicht zurückgesetzt werden');
      }
    } catch (e: any) {
      error.value = e.message;
      throw e;
    } finally {
      isLoading.value = false;
    }
  }

  return {
    // State
    user,
    isAuthenticated,
    isAdmin,
    isLoading,
    error,

    // Actions
    login,
    logout,
    refreshAccessToken,
    requestPasswordReset,
    confirmPasswordReset,
    initAuth,
  };
}

// Initialize on module load
initAuth();
