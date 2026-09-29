<script setup lang="ts">
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import logo from '@/assets/knirpsenstadt-logo.png';

const route = useRoute();
const router = useRouter();
const token = typeof route.query.token === 'string' ? route.query.token : '';
const password = ref('');
const confirm = ref('');
const error = ref('');
const saving = ref(false);

async function submit() {
  error.value = '';
  if (password.value.length < 8) {
    error.value = 'Das Passwort muss mindestens 8 Zeichen haben.';
    return;
  }
  if (password.value !== confirm.value) {
    error.value = 'Die Passwörter stimmen nicht überein.';
    return;
  }
  saving.value = true;
  try {
    await api.setInvitationPassword(token, password.value);
    await router.replace({ name: 'login', query: { eingeladen: '1' } });
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Passwort konnte nicht gesetzt werden.';
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background px-4 py-8">
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <img :src="logo" alt="" class="mx-auto mb-4 h-20 w-20 rounded-full object-cover" />
        <h1 class="text-3xl font-bold text-primary">Kita Knirpsenstadt</h1>
        <p class="mt-2 text-muted-foreground">Beitragsverwaltung</p>
      </div>
      <div class="rounded-2xl border bg-card p-8 shadow-lg">
        <h2 class="mb-4 text-xl font-semibold">Passwort setzen</h2>
        <p v-if="!token" class="text-destructive">Der Einladungslink ist ungültig.</p>
        <form v-else class="space-y-4" @submit.prevent="submit">
          <p class="text-sm text-muted-foreground">
            Wähle ein Passwort mit mindestens 8 Zeichen. Damit meldest du dich danach an.
          </p>
          <label class="block text-sm font-medium">Passwort
            <input v-model="password" type="password" autocomplete="new-password" required minlength="8"
              class="mt-1 w-full rounded-xl border border-input bg-card px-3 py-2 text-foreground" />
          </label>
          <label class="block text-sm font-medium">Passwort wiederholen
            <input v-model="confirm" type="password" autocomplete="new-password" required minlength="8"
              class="mt-1 w-full rounded-xl border border-input bg-card px-3 py-2 text-foreground" />
          </label>
          <p v-if="error" class="text-sm text-destructive" role="alert">{{ error }}</p>
          <button type="submit" :disabled="saving"
            class="w-full rounded-xl bg-primary px-4 py-2.5 text-primary-foreground disabled:opacity-50">
            {{ saving ? 'Speichern...' : 'Passwort setzen' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>
