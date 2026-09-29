<script setup lang="ts">
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { Loader2 } from 'lucide-vue-next';
import logo from '@/assets/knirpsenstadt-logo.png';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const email = ref('');
const password = ref('');

async function handleSubmit() {
  const success = await authStore.login(email.value, password.value);
  if (success) {
    const redirect = route.query.redirect as string;
    router.push(redirect || (authStore.isParent ? '/familie' : authStore.isParentWork ? '/elternstunden' : '/'));
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background px-4 py-8">
    <div class="w-full max-w-md">
      <!-- Header -->
      <div class="text-center mb-8">
        <img :src="logo" alt="" class="mx-auto mb-4 h-20 w-20 rounded-full object-cover shadow-sm" />
        <h1 class="text-3xl font-bold text-primary">Kita Knirpsenstadt</h1>
        <p class="mt-2 text-muted-foreground">Beitragsverwaltung</p>
      </div>

      <!-- Login Card -->
      <div class="rounded-2xl border bg-card p-8 text-card-foreground shadow-lg">
        <h2 class="text-xl font-semibold mb-6">Anmelden</h2>

        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- Email -->
          <div>
            <label for="email" class="mb-1 block text-sm font-medium text-foreground">
              E-Mail
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              required
              autocomplete="email"
              class="w-full rounded-xl border border-input bg-card px-3 py-2 text-foreground outline-none transition-shadow focus:ring-2 focus:ring-ring"
              placeholder="name@knirpsenstadt.de"
            />
          </div>

          <!-- Password -->
          <div>
            <label for="password" class="mb-1 block text-sm font-medium text-foreground">
              Passwort
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              required
              autocomplete="current-password"
              class="w-full rounded-xl border border-input bg-card px-3 py-2 text-foreground outline-none transition-shadow focus:ring-2 focus:ring-ring"
              placeholder="••••••••"
            />
          </div>

          <!-- Error message -->
          <div v-if="authStore.error" class="rounded-xl border border-destructive/30 bg-destructive/10 p-3">
            <p class="text-sm text-destructive dark:text-destructive-foreground">{{ authStore.error }}</p>
          </div>

          <!-- Submit button -->
          <button
            type="submit"
            :disabled="authStore.isLoading"
            class="flex w-full items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2.5 font-medium text-primary-foreground transition-colors hover:bg-brand-800 focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-brand-200"
          >
            <Loader2 v-if="authStore.isLoading" class="h-4 w-4 animate-spin" />
            {{ authStore.isLoading ? 'Anmelden...' : 'Anmelden' }}
          </button>

        </form>
      </div>

    </div>
  </div>
</template>
