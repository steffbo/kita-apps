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
    router.push(redirect || (authStore.isParent ? '/familie' : authStore.isParentWork ? '/elternstunden' : authStore.isLeitung ? '/kinder' : '/'));
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background px-4 py-8">
    <div class="w-full max-w-md">
      <!-- Header -->
      <div class="text-center mb-8">
        <img :src="logo" alt="" class="mx-auto mb-4 h-20 w-20 rounded-full object-cover shadow-sm" />
        <h1 class="text-3xl font-semibold text-primary">Kita Knirpsenstadt</h1>
        <p class="mt-2 text-muted-foreground">Beitragsverwaltung</p>
      </div>

      <!-- Login Card -->
      <div class="card p-6 sm:p-8">
        <h2 class="text-xl font-semibold mb-6">Anmelden</h2>
        <p v-if="route.query.eingeladen === '1'"
          class="note note-success mb-4">
          Dein Passwort ist gesetzt. Du kannst dich jetzt anmelden.
        </p>

        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- Email -->
          <div>
            <label for="email" class="label">
              E-Mail
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              required
              autocomplete="email"
              class="input"
              placeholder="name@knirpsenstadt.de"
            />
          </div>

          <!-- Password -->
          <div>
            <label for="password" class="label">
              Passwort
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              required
              autocomplete="current-password"
              class="input"
              placeholder="••••••••"
            />
          </div>

          <!-- Error message -->
          <div v-if="authStore.error" class="note note-danger">
            <p>{{ authStore.error }}</p>
          </div>

          <!-- Submit button -->
          <button
            type="submit"
            :disabled="authStore.isLoading"
            class="btn btn-primary w-full"
          >
            <Loader2 v-if="authStore.isLoading" class="h-4 w-4 animate-spin" />
            {{ authStore.isLoading ? 'Anmelden...' : 'Anmelden' }}
          </button>

        </form>
      </div>

    </div>
  </div>
</template>
