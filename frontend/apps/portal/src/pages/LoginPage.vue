<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { ArrowRight, KeyRound, Mail, Sprout } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const email = ref('');
const password = ref('');
const errorMessage = ref('');
const isSubmitting = ref(false);

async function submit() {
  errorMessage.value = '';
  isSubmitting.value = true;

  try {
    await authStore.login(email.value, password.value);
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    router.push(redirect);
  } catch {
    errorMessage.value = 'E-Mail oder Passwort stimmen nicht.';
  } finally {
    isSubmitting.value = false;
  }
}
</script>

<template>
  <main class="min-h-[100dvh] bg-background">
    <div class="mx-auto grid min-h-[100dvh] max-w-7xl grid-cols-1 lg:grid-cols-[minmax(0,0.95fr)_minmax(420px,0.7fr)]">
      <section class="flex min-h-[46dvh] flex-col justify-between px-5 py-6 sm:px-8 lg:min-h-[100dvh] lg:px-10 lg:py-10">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-soft">
            <Sprout class="h-5 w-5" :stroke-width="1.8" />
          </div>
          <div>
            <p class="text-sm font-semibold">Kita Knirpsenstadt</p>
            <p class="text-xs text-muted-foreground">Portal</p>
          </div>
        </div>

        <div class="max-w-[620px] py-12 lg:py-0">
          <p class="mb-4 text-sm font-semibold uppercase tracking-[0.18em] text-primary">Hallo</p>
          <h1 class="text-4xl font-semibold leading-[1.05] tracking-normal text-foreground sm:text-5xl">
            Willkommen im Kita-Portal.
          </h1>
          <p class="mt-5 max-w-[54ch] text-base leading-7 text-muted-foreground">
            Bitte melde dich mit deiner E-Mail-Adresse und deinem Passwort an.
          </p>
        </div>

        <div class="hidden items-center gap-3 text-xs text-muted-foreground lg:flex">
          <span class="h-px w-10 bg-border"></span>
          <span>portal.knirpsenstadt.de</span>
        </div>
      </section>

      <section class="flex items-center px-5 pb-10 sm:px-8 lg:min-h-[100dvh] lg:px-10 lg:py-10">
        <form class="w-full rounded-lg border bg-surface p-5 shadow-soft sm:p-7" @submit.prevent="submit">
          <div class="mb-7">
            <h2 class="text-2xl font-semibold tracking-normal">Anmeldung</h2>
            <p class="mt-2 text-sm leading-6 text-muted-foreground">
              Nutze die Adresse aus deiner Einladung.
            </p>
          </div>

          <div class="grid gap-4">
            <label class="field">
              <span class="field-label">E-Mail</span>
              <span class="relative">
                <Mail class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" :stroke-width="1.8" />
                <input
                  v-model="email"
                  class="field-input w-full pl-9"
                  type="email"
                  autocomplete="email"
                  required
                  placeholder="name@knirpsenstadt.de"
                />
              </span>
            </label>

            <label class="field">
              <span class="field-label">Passwort</span>
              <span class="relative">
                <KeyRound class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" :stroke-width="1.8" />
                <input
                  v-model="password"
                  class="field-input w-full pl-9"
                  type="password"
                  autocomplete="current-password"
                  required
                  minlength="8"
                  placeholder="Mindestens 8 Zeichen"
                />
              </span>
            </label>
          </div>

          <p v-if="errorMessage" class="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {{ errorMessage }}
          </p>

          <button class="primary-button mt-6 w-full justify-between" type="submit" :disabled="isSubmitting">
            <span>{{ isSubmitting ? 'Anmeldung läuft' : 'Anmelden' }}</span>
            <ArrowRight class="h-4 w-4" :stroke-width="1.8" />
          </button>

          <div class="mt-5 flex justify-center">
            <RouterLink class="text-sm font-medium text-primary underline-offset-4 hover:underline" :to="{ name: 'password-reset' }">
              Passwort vergessen
            </RouterLink>
          </div>
        </form>
      </section>
    </div>
  </main>
</template>
