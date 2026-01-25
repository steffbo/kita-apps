import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { api } from '@/api';
import type { User, TokenPair } from '@/api/types';

const ACCESS_TOKEN_KEY = 'fees_access_token';
const REFRESH_TOKEN_KEY = 'fees_refresh_token';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const accessToken = ref<string | null>(localStorage.getItem(ACCESS_TOKEN_KEY));
  const refreshToken = ref<string | null>(localStorage.getItem(REFRESH_TOKEN_KEY));
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  const isAuthenticated = computed(() => !!accessToken.value);
  const isAdmin = computed(() => user.value?.role === 'ADMIN');

  // Initialize API client with stored tokens and callbacks
  if (accessToken.value) {
    api.setAccessToken(accessToken.value);
  }
  if (refreshToken.value) {
    api.setRefreshToken(refreshToken.value);
  }

  // Set up callback for when API client refreshes tokens
  api.setOnTokenRefreshed((tokens) => {
    accessToken.value = tokens.accessToken;
    refreshToken.value = tokens.refreshToken;
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
  });

  // Set up callback for when auth completely fails
  api.setOnAuthFailed(() => {
    clearTokens();
  });

  function setTokens(tokens: TokenPair) {
    accessToken.value = tokens.accessToken;
    refreshToken.value = tokens.refreshToken;
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    api.setAccessToken(tokens.accessToken);
    api.setRefreshToken(tokens.refreshToken);
  }

  function clearTokens() {
    accessToken.value = null;
    refreshToken.value = null;
    user.value = null;
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    api.setAccessToken(null);
    api.setRefreshToken(null);
  }

  async function login(email: string, password: string) {
    isLoading.value = true;
    error.value = null;

    try {
      const tokens = await api.login({ email, password });
      setTokens(tokens);
      await fetchUser();
      return true;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Login fehlgeschlagen';
      clearTokens();
      return false;
    } finally {
      isLoading.value = false;
    }
  }

  async function logout() {
    try {
      await api.logout();
    } catch {
      // Ignore logout errors
    } finally {
      clearTokens();
    }
  }

  async function fetchUser() {
    if (!accessToken.value) return;

    try {
      user.value = await api.me();
    } catch (e) {
      if (e instanceof Error && e.message === 'Unauthorized') {
        await tryRefresh();
      }
    }
  }

  async function tryRefresh(): Promise<boolean> {
    if (!refreshToken.value) {
      clearTokens();
      return false;
    }

    try {
      const tokens = await api.refresh(refreshToken.value);
      setTokens(tokens);
      await fetchUser();
      return true;
    } catch {
      clearTokens();
      return false;
    }
  }

  async function initialize() {
    if (accessToken.value) {
      await fetchUser();
    }
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    isLoading.value = true;
    error.value = null;

    try {
      await api.changePassword(currentPassword, newPassword);
      return true;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Passwort konnte nicht geändert werden';
      throw e;
    } finally {
      isLoading.value = false;
    }
  }

  return {
    user,
    isAuthenticated,
    isAdmin,
    isLoading,
    error,
    login,
    logout,
    fetchUser,
    tryRefresh,
    initialize,
    changePassword,
  };
});
