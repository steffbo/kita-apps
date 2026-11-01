import { defineStore } from 'pinia';
import { ref, computed, watch } from 'vue';
import { api } from '@/api';
import type { User } from '@/api/types';

// Tokens used to live in localStorage; remove leftovers from older versions.
localStorage.removeItem('fees_access_token');
localStorage.removeItem('fees_refresh_token');

// Survives a reload of the tab (not the browser session): the marker names the
// user an admin is impersonating; the tokens themselves stay in memory.
const IMPERSONATION_KEY = 'fees_impersonating';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  // Access token in memory only. The session itself is the httpOnly refresh
  // cookie, so a reload restores it via initialize().
  const accessToken = ref<string | null>(null);
  // The admin behind an impersonated session; `user` is then the target.
  const impersonator = ref<User | null>(null);
  const initialized = ref(false);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  const isAuthenticated = computed(() => !!accessToken.value);
  const isImpersonating = computed(() => !!impersonator.value);
  const isAdmin = computed(() => user.value?.role === 'ADMIN');
  const isBoard = computed(() => user.value?.role === 'BOARD');
  const isParent = computed(() => user.value?.role === 'PARENT');
  const isParentWork = computed(() => user.value?.role === 'PARENT_WORK');
  const isLeitung = computed(() => user.value?.role === 'LEITUNG');
  /** Beiträge, Einstufungen, Bankabgleich, Notizen; never for Leitung. */
  const canAccessFees = computed(() => isAdmin.value || isBoard.value);
  /** Vorstand sieht alles, ändert aber nichts (the API answers writes with 403). */
  const isReadOnly = computed(() => isBoard.value);
  // Hides every [data-write] control, including those in teleported dialogs (main.css).
  watch(isReadOnly, (readOnly) => {
    document.documentElement.classList.toggle('read-only', readOnly);
  }, { immediate: true });
  /** Kinder, Eltern, Haushalte, Mitglieder (Leitung ohne Beitrags- und Einkommensdaten). */
  const canAccessMasterData = computed(() => canAccessFees.value || isLeitung.value);
  const canAccessParentWork = computed(() => isAdmin.value || isBoard.value || isParentWork.value || isLeitung.value);
  /** Regeln, Import, Soll, Bearbeiten/Stornieren/Prüfen; Leitung sieht nur den Stand und erfasst Stunden. */
  const canManageParentWork = computed(() => isAdmin.value || isParentWork.value);
  /** Verwaltungsseite: Vorstand sieht dort nur die Regeln. */
  const canViewParentWorkAdmin = computed(() => canManageParentWork.value || isBoard.value);

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
    impersonator.value = null;
    sessionStorage.removeItem(IMPERSONATION_KEY);
    api.setTokenExchange(null);
    api.setAccessToken(null);
  }

  // The refresh cookie always belongs to the admin, so every refresh while
  // impersonating has to be traded for a fresh token of the target user.
  async function exchangeForImpersonation(adminToken: string): Promise<string | null> {
    const targetId = sessionStorage.getItem(IMPERSONATION_KEY);
    if (!targetId) return null;
    try {
      const result = await api.impersonateUser(targetId, adminToken);
      user.value = result.user;
      return result.accessToken;
    } catch {
      return null;
    }
  }

  function homePath(target: User): string {
    if (target.role === 'PARENT') return '/familie';
    if (target.role === 'PARENT_WORK') return '/elternstunden';
    if (target.role === 'LEITUNG') return '/kinder';
    return '/';
  }

  // Full reload: no data of the previous identity stays in memory, and
  // initialize() restores (or ends) the mode from the marker.
  function reloadAt(path: string) {
    window.location.assign(import.meta.env.BASE_URL.replace(/\/$/, '') + path);
  }

  /** Validates the request, then reloads as the target user. Throws the backend's message. */
  async function startImpersonation(userId: string) {
    const result = await api.impersonateUser(userId);
    sessionStorage.setItem(IMPERSONATION_KEY, userId);
    reloadAt(homePath(result.user));
  }

  function stopImpersonation() {
    sessionStorage.removeItem(IMPERSONATION_KEY);
    reloadAt('/benutzer');
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
    if (!(await api.tryRefreshToken())) return;
    await fetchUser();

    const targetId = sessionStorage.getItem(IMPERSONATION_KEY);
    if (!targetId || !user.value) return;
    const admin = user.value;
    try {
      const result = await api.impersonateUser(targetId);
      impersonator.value = admin;
      setAccessToken(result.accessToken);
      user.value = result.user;
      api.setTokenExchange(exchangeForImpersonation);
    } catch {
      // Target gone or deactivated: stay signed in as the admin.
      sessionStorage.removeItem(IMPERSONATION_KEY);
    }
  }

  return {
    user,
    impersonator,
    isImpersonating,
    isAuthenticated,
    isAdmin,
    isBoard,
    isReadOnly,
    isParent,
    isParentWork,
    isLeitung,
    canAccessFees,
    canAccessMasterData,
    canAccessParentWork,
    canManageParentWork,
    canViewParentWorkAdmin,
    isLoading,
    error,
    login,
    logout,
    fetchUser,
    changePassword,
    initialize,
    startImpersonation,
    stopImpersonation,
  };
});
