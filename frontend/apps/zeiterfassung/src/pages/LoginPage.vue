<script setup lang="ts">
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuth } from '@kita/shared';

const router = useRouter();
const route = useRoute();
const { login, isLoading, error } = useAuth();

const email = ref('');
const password = ref('');

async function handleSubmit() {
  try {
    await login(email.value, password.value);
    const redirect = (route.query.redirect as string) || '/';
    router.push(redirect);
  } catch {
    // Error is handled by useAuth
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 px-4">
    <div class="w-full max-w-md">
      <div class="bg-white rounded-lg shadow-lg p-8">
        <div class="text-center mb-8">
          <h1 class="text-2xl font-bold text-stone-900">Zeiterfassung</h1>
          <p class="text-stone-600 mt-2">Kita Knirpsenstadt</p>
        </div>

        <form @submit.prevent="handleSubmit" class="space-y-6">
          <div v-if="error" class="bg-red-50 text-red-700 p-3 rounded-md text-sm">
            {{ error }}
          </div>

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

          <div>
            <label for="password" class="block text-sm font-medium text-stone-700 mb-1">
              Passwort
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              required
              class="w-full px-3 py-2 border border-stone-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
              placeholder="••••••••"
            />
          </div>

          <button
            type="submit"
            :disabled="isLoading"
            class="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <span v-if="isLoading">Wird angemeldet...</span>
            <span v-else>Anmelden</span>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>
