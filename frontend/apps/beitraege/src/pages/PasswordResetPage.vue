<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { Loader2 } from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

// Mode: 'request' for requesting reset, 'confirm' for setting new password
const token = computed(() => route.query.token as string | undefined);
const mode = computed(() => token.value ? 'confirm' : 'request');

const email = ref('');
const newPassword = ref('');
const confirmNewPassword = ref('');
const submitted = ref(false);
const resetSuccess = ref(false);
const localError = ref<string | null>(null);

const passwordMismatch = computed(() => {
  return newPassword.value !== confirmNewPassword.value && confirmNewPassword.value.length > 0;
});

const passwordTooShort = computed(() => {
  return newPassword.value.length > 0 && newPassword.value.length < 8;
});

async function handleRequestReset() {
  localError.value = null;
  try {
    await authStore.requestPasswordReset(email.value);
    submitted.value = true;
  } catch {
    // Error handled by store
  }
}

async function handleConfirmReset() {
  localError.value = null;

  if (newPassword.value !== confirmNewPassword.value) {
    localError.value = 'Passwörter stimmen nicht überein';
    return;
  }

  if (newPassword.value.length < 8) {
    localError.value = 'Passwort muss mindestens 8 Zeichen lang sein';
    return;
  }

  try {
    await authStore.confirmPasswordReset(token.value!, newPassword.value);
    resetSuccess.value = true;
  } catch {
    // Error handled by store
  }
}

function goToLogin() {
  router.push('/login');
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-md">
      <!-- Header -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-primary">Kita Knirpsenstadt</h1>
        <p class="text-gray-600 mt-2">Beitragsverwaltung</p>
      </div>

      <!-- Card -->
      <div class="bg-white rounded-xl shadow-lg p-8">
        <h2 class="text-xl font-semibold mb-6">
          {{ mode === 'confirm' ? 'Neues Passwort festlegen' : 'Passwort zurücksetzen' }}
        </h2>

        <!-- Request Reset: Success Message -->
        <div v-if="mode === 'request' && submitted" class="text-center">
          <div class="bg-green-50 border border-green-200 text-green-700 p-4 rounded-lg mb-4">
            Falls ein Account mit dieser E-Mail existiert, wurde eine E-Mail zum Zurücksetzen des Passworts gesendet.
          </div>
          <router-link to="/login" class="text-primary hover:text-primary/80 font-medium">
            Zurück zum Login
          </router-link>
        </div>

        <!-- Confirm Reset: Success Message -->
        <div v-else-if="mode === 'confirm' && resetSuccess" class="text-center">
          <div class="bg-green-50 border border-green-200 text-green-700 p-4 rounded-lg mb-4">
            Dein Passwort wurde erfolgreich zurückgesetzt. Du kannst dich jetzt mit deinem neuen Passwort anmelden.
          </div>
          <button
            @click="goToLogin"
            class="text-primary hover:text-primary/80 font-medium"
          >
            Zum Login
          </button>
        </div>

        <!-- Request Reset Form -->
        <form v-else-if="mode === 'request'" @submit.prevent="handleRequestReset" class="space-y-4">
          <div v-if="authStore.error" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ authStore.error }}</p>
          </div>

          <p class="text-sm text-gray-600">
            Gib deine E-Mail-Adresse ein. Du erhältst einen Link zum Zurücksetzen deines Passworts.
          </p>

          <div>
            <label for="email" class="block text-sm font-medium text-gray-700 mb-1">
              E-Mail
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              required
              autocomplete="email"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-shadow"
              placeholder="name@knirpsenstadt.de"
            />
          </div>

          <button
            type="submit"
            :disabled="authStore.isLoading"
            class="w-full py-2.5 px-4 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 focus:ring-2 focus:ring-primary focus:ring-offset-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            <Loader2 v-if="authStore.isLoading" class="h-4 w-4 animate-spin" />
            {{ authStore.isLoading ? 'Wird gesendet...' : 'Link senden' }}
          </button>

          <div class="text-center">
            <router-link to="/login" class="text-sm text-primary hover:text-primary/80">
              Zurück zum Login
            </router-link>
          </div>
        </form>

        <!-- Confirm Reset Form -->
        <form v-else @submit.prevent="handleConfirmReset" class="space-y-4">
          <div v-if="authStore.error || localError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ authStore.error || localError }}</p>
          </div>

          <p class="text-sm text-gray-600">
            Gib dein neues Passwort ein.
          </p>

          <div>
            <label for="newPassword" class="block text-sm font-medium text-gray-700 mb-1">
              Neues Passwort
            </label>
            <input
              id="newPassword"
              v-model="newPassword"
              type="password"
              required
              minlength="8"
              class="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-shadow"
              :class="passwordTooShort ? 'border-red-300' : 'border-gray-300'"
              placeholder="Mindestens 8 Zeichen"
            />
            <p v-if="passwordTooShort" class="mt-1 text-sm text-red-600">
              Mindestens 8 Zeichen erforderlich
            </p>
          </div>

          <div>
            <label for="confirmPassword" class="block text-sm font-medium text-gray-700 mb-1">
              Passwort bestätigen
            </label>
            <input
              id="confirmPassword"
              v-model="confirmNewPassword"
              type="password"
              required
              class="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-shadow"
              :class="passwordMismatch ? 'border-red-300' : 'border-gray-300'"
              placeholder="Passwort wiederholen"
            />
            <p v-if="passwordMismatch" class="mt-1 text-sm text-red-600">
              Passwörter stimmen nicht überein
            </p>
          </div>

          <button
            type="submit"
            :disabled="authStore.isLoading || passwordMismatch || passwordTooShort"
            class="w-full py-2.5 px-4 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 focus:ring-2 focus:ring-primary focus:ring-offset-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            <Loader2 v-if="authStore.isLoading" class="h-4 w-4 animate-spin" />
            {{ authStore.isLoading ? 'Wird gespeichert...' : 'Passwort speichern' }}
          </button>

          <div class="text-center">
            <router-link to="/login" class="text-sm text-primary hover:text-primary/80">
              Zurück zum Login
            </router-link>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
