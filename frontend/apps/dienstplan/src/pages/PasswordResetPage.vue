<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '@kita/shared';

const route = useRoute();
const router = useRouter();
const { requestPasswordReset, confirmPasswordReset, isLoading, error } = useAuth();

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
    await requestPasswordReset(email.value);
    submitted.value = true;
  } catch {
    // Error handled by useAuth
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
    await confirmPasswordReset(token.value!, newPassword.value);
    resetSuccess.value = true;
  } catch {
    // Error handled by useAuth
  }
}

function goToLogin() {
  router.push('/login');
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 px-4">
    <div class="w-full max-w-md">
      <div class="bg-white rounded-lg shadow-lg p-8">
        <div class="text-center mb-8">
          <h1 class="text-2xl font-bold text-stone-900">
            {{ mode === 'confirm' ? 'Neues Passwort festlegen' : 'Passwort zurücksetzen' }}
          </h1>
          <p class="text-stone-600 mt-2">Kita Knirpsenstadt</p>
        </div>

        <!-- Request Reset: Success Message -->
        <div v-if="mode === 'request' && submitted" class="text-center">
          <div class="bg-green-50 text-green-700 p-4 rounded-md mb-4">
            Falls ein Account mit dieser E-Mail existiert, wurde eine E-Mail zum Zurücksetzen des Passworts gesendet.
          </div>
          <router-link to="/login" class="text-green-600 hover:text-green-700 font-medium">
            Zurück zum Login
          </router-link>
        </div>

        <!-- Confirm Reset: Success Message -->
        <div v-else-if="mode === 'confirm' && resetSuccess" class="text-center">
          <div class="bg-green-50 text-green-700 p-4 rounded-md mb-4">
            Dein Passwort wurde erfolgreich zurückgesetzt. Du kannst dich jetzt mit deinem neuen Passwort anmelden.
          </div>
          <button
            @click="goToLogin"
            class="text-green-600 hover:text-green-700 font-medium"
          >
            Zum Login
          </button>
        </div>

        <!-- Request Reset Form -->
        <form v-else-if="mode === 'request'" @submit.prevent="handleRequestReset" class="space-y-6">
          <div v-if="error" class="bg-red-50 text-red-700 p-3 rounded-md text-sm">
            {{ error }}
          </div>

          <p class="text-sm text-stone-600">
            Gib deine E-Mail-Adresse ein. Du erhältst einen Link zum Zurücksetzen deines Passworts.
          </p>

          <div>
            <label for="email" class="block text-sm font-medium text-stone-700 mb-1">
              E-Mail
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              required
              class="w-full px-3 py-2 border border-stone-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
              placeholder="name@knirpsenstadt.de"
            />
          </div>

          <button
            type="submit"
            :disabled="isLoading"
            class="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <span v-if="isLoading">Wird gesendet...</span>
            <span v-else>Link senden</span>
          </button>

          <div class="text-center">
            <router-link to="/login" class="text-sm text-green-600 hover:text-green-700">
              Zurück zum Login
            </router-link>
          </div>
        </form>

        <!-- Confirm Reset Form -->
        <form v-else @submit.prevent="handleConfirmReset" class="space-y-6">
          <div v-if="error || localError" class="bg-red-50 text-red-700 p-3 rounded-md text-sm">
            {{ error || localError }}
          </div>

          <p class="text-sm text-stone-600">
            Gib dein neues Passwort ein.
          </p>

          <div>
            <label for="newPassword" class="block text-sm font-medium text-stone-700 mb-1">
              Neues Passwort
            </label>
            <input
              id="newPassword"
              v-model="newPassword"
              type="password"
              required
              minlength="8"
              class="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
              :class="passwordTooShort ? 'border-red-300' : 'border-stone-300'"
              placeholder="Mindestens 8 Zeichen"
            />
            <p v-if="passwordTooShort" class="mt-1 text-sm text-red-600">
              Mindestens 8 Zeichen erforderlich
            </p>
          </div>

          <div>
            <label for="confirmPassword" class="block text-sm font-medium text-stone-700 mb-1">
              Passwort bestätigen
            </label>
            <input
              id="confirmPassword"
              v-model="confirmNewPassword"
              type="password"
              required
              class="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
              :class="passwordMismatch ? 'border-red-300' : 'border-stone-300'"
              placeholder="Passwort wiederholen"
            />
            <p v-if="passwordMismatch" class="mt-1 text-sm text-red-600">
              Passwörter stimmen nicht überein
            </p>
          </div>

          <button
            type="submit"
            :disabled="isLoading || passwordMismatch || passwordTooShort"
            class="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <span v-if="isLoading">Wird gespeichert...</span>
            <span v-else>Passwort speichern</span>
          </button>

          <div class="text-center">
            <router-link to="/login" class="text-sm text-green-600 hover:text-green-700">
              Zurück zum Login
            </router-link>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
