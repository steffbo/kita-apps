<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink } from 'vue-router';
import { ArrowLeft, Mail } from 'lucide-vue-next';

const email = ref('');
const isSubmitted = ref(false);

function submit() {
  isSubmitted.value = true;
}
</script>

<template>
  <main class="flex min-h-[100dvh] items-center justify-center bg-background px-5 py-10">
    <section class="w-full max-w-[460px] rounded-lg border bg-surface p-6 shadow-soft">
      <RouterLink class="mb-7 inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground" :to="{ name: 'login' }">
        <ArrowLeft class="h-4 w-4" :stroke-width="1.8" />
        Zur Anmeldung
      </RouterLink>

      <h1 class="text-2xl font-semibold tracking-normal">Passwort zurücksetzen</h1>
      <p class="mt-2 text-sm leading-6 text-muted-foreground">
        Wenn ein Konto existiert, senden wir einen Link zum Zurücksetzen.
      </p>

      <form v-if="!isSubmitted" class="mt-6 grid gap-5" @submit.prevent="submit">
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
        <button class="primary-button w-full" type="submit">Link anfordern</button>
      </form>

      <p v-else class="mt-6 rounded-lg border bg-muted px-3 py-3 text-sm leading-6 text-muted-foreground">
        Bitte prüfe dein Postfach.
      </p>
    </section>
  </main>
</template>
