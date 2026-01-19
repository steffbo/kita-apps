<script setup lang="ts">
import { ref } from 'vue';
import { useAuth } from '@kita/shared';

const { requestPasswordReset, isLoading, error } = useAuth();

const email = ref('');
const submitted = ref(false);

async function handleSubmit() {
  try {
    await requestPasswordReset(email.value);
    submitted.value = true;
  } catch {
    // Error handled by useAuth
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 px-4">
    <div class="w-full max-w-md">
      <div class="bg-white rounded-lg shadow-lg p-8">
        <div class="text-center mb-8">
          <h1 class="text-2xl font-bold text-stone-900">Passwort zurücksetzen</h1>
          <p class="text-stone-600 mt-2">Kita Knirpsenstadt</p>
        </div>

        <div v-if="submitted" class="text-center">
          <div class="bg-green-50 text-green-700 p-4 rounded-md mb-4">
            Falls ein Account mit dieser E-Mail existiert, wurde eine E-Mail zum Zurücksetzen des Passworts gesendet.
          </div>
          <router-link to="/login" class="text-green-600 hover:text-green-700 font-medium">
            Zurück zum Login
          </router-link>
        </div>

        <form v-else @submit.prevent="handleSubmit" class="space-y-6">
          <div v-if="error" class="bg-red-50 text-red-700 p-3 rounded-md text-sm">
            {{ error }}
          </div>

          <p class="text-sm text-stone-600">
            Geben Sie Ihre E-Mail-Adresse ein. Sie erhalten einen Link zum Zurücksetzen Ihres Passworts.
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
      </div>
    </div>
  </div>
</template>
