import { computed, ref } from 'vue';
import { defineStore } from 'pinia';
import type { PortalRole } from '@/lib/modules';

interface PortalUser {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  roles: PortalRole[];
}

interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  user: PortalUser;
}

const accessTokenKey = 'kita.portal.accessToken';
const refreshTokenKey = 'kita.portal.refreshToken';
const userKey = 'kita.portal.user';

function readStoredUser(): PortalUser | null {
  const stored = localStorage.getItem(userKey);
  if (!stored) {
    return null;
  }

  try {
    return JSON.parse(stored) as PortalUser;
  } catch {
    localStorage.removeItem(userKey);
    return null;
  }
}

export const useAuthStore = defineStore('portal-auth', () => {
  const accessToken = ref<string | null>(localStorage.getItem(accessTokenKey));
  const refreshToken = ref<string | null>(localStorage.getItem(refreshTokenKey));
  const user = ref<PortalUser | null>(readStoredUser());
  const isAuthenticated = computed(() => Boolean(accessToken.value && user.value));
  const displayName = computed(() => {
    if (!user.value) {
      return '';
    }

    const fullName = [user.value.firstName, user.value.lastName].filter(Boolean).join(' ');
    return fullName || user.value.email;
  });

  async function login(email: string, password: string) {
    const response = await fetch('/api/portal/v1/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      throw new Error('Anmeldung fehlgeschlagen');
    }

    const body = (await response.json()) as LoginResponse;
    accessToken.value = body.accessToken;
    refreshToken.value = body.refreshToken;
    user.value = body.user;

    localStorage.setItem(accessTokenKey, body.accessToken);
    localStorage.setItem(refreshTokenKey, body.refreshToken);
    localStorage.setItem(userKey, JSON.stringify(body.user));
  }

  function logout() {
    accessToken.value = null;
    refreshToken.value = null;
    user.value = null;
    localStorage.removeItem(accessTokenKey);
    localStorage.removeItem(refreshTokenKey);
    localStorage.removeItem(userKey);
  }

  function hasAnyRole(roles: PortalRole[]) {
    if (!user.value) {
      return false;
    }
    return roles.some((role) => user.value?.roles.includes(role));
  }

  return {
    accessToken,
    displayName,
    hasAnyRole,
    isAuthenticated,
    login,
    logout,
    refreshToken,
    user,
  };
});
