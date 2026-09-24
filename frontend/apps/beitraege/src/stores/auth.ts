import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { api } from '@/api';
import type { User } from '@/api/types';

// Tokens used to live in localStorage; remove leftovers from older versions.
localStorage.removeItem('fees_access_token');
localStorage.removeItem('fees_refresh_token');

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  // Access token in memory only. The session itself is the httpOnly refresh
  // cookie, so a reload restores it via initialize().
  const accessToken = ref<string | null>(null);
  const initialized = ref(false);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  const isAuthenticated = computed(() => !!accessToken.value);
  const isAdmin = computed(() => user.value?.role === 'ADMIN');

  api.setOnTokenRefreshed((token) => {
    accessToken.value = token;
  });

  api.setOnAuthFailed(() => {
    clearSession();
  });

  function setAccessToken(token: string) {
    accessToken.value = token;
    api.setAccessToken(token);
  }

  function clearSession() {
    accessToken.value = null;
    user.value = null;
    api.setAccessToken(null);
  }

  async function login(email: string, password: string) {
    isLoading.value = true;
    error.value = null;

    try {
      const result = await api.login({ email, password });
      setAccessToken(result.accessToken);
      user.value = result.user;
      return true;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Login fehlgeschlagen';
      clearSession();
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
      clearSession();
    }
  }

  async function fetchUser() {
    if (!accessToken.value) return;
    try {
      user.value = await api.me();
    } catch {
      // A failed refresh already cleared the session via onAuthFailed.
    }
  }

  // Changing the password ends all other sessions; the response carries ours.
  async function changePassword(currentPassword: string, newPassword: string) {
    const result = await api.changePassword({ currentPassword, newPassword });
    setAccessToken(result.accessToken);
  }

  /** Restores the session from the refresh cookie once per page load. */
  async function initialize() {
    if (initialized.value) return;
    initialized.value = true;
    if (await api.tryRefreshToken()) {
      await fetchUser();
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
    changePassword,
    initialize,
  };
});
