<script setup lang="ts">
import { ref } from 'vue';
import { Check, Clock3, Plus } from 'lucide-vue-next';

const category = ref('Garten');
const submitted = ref(false);

function submit() {
  submitted.value = true;
}
</script>

<template>
  <section class="mx-auto max-w-7xl px-5 py-8 sm:px-8 lg:px-10 lg:py-10">
    <div class="mb-8 grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-end">
      <div>
        <p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">Elternstunden</p>
        <h1 class="mt-3 text-3xl font-semibold tracking-normal sm:text-4xl">Einreichung und Übersicht</h1>
      </div>
      <div class="grid grid-cols-2 gap-3">
        <div class="rounded-lg border bg-surface p-4">
          <p class="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">Genehmigt</p>
          <p class="mt-3 text-2xl font-semibold tabular-nums">0,0 h</p>
        </div>
        <div class="rounded-lg border bg-surface p-4">
          <p class="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">Offen</p>
          <p class="mt-3 text-2xl font-semibold tabular-nums">0,0 h</p>
        </div>
      </div>
    </div>

    <div class="grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
      <form class="rounded-lg border bg-surface p-5 shadow-soft" @submit.prevent="submit">
        <div class="mb-5 flex items-center justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold">Eintrag erfassen</h2>
            <p class="mt-1 text-sm text-muted-foreground">Keine Datei-Uploads im MVP.</p>
          </div>
          <Plus class="h-5 w-5 text-primary" :stroke-width="1.8" />
        </div>

        <div class="grid gap-4">
          <label class="field">
            <span class="field-label">Kind</span>
            <select class="field-input w-full" required>
              <option value="">Kind wählen</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Datum</span>
            <input class="field-input" type="date" required />
          </label>
          <label class="field">
            <span class="field-label">Dauer</span>
            <input class="field-input" type="number" min="0.25" step="0.25" placeholder="2,5" required />
          </label>
          <label class="field">
            <span class="field-label">Kategorie</span>
            <select v-model="category" class="field-input">
              <option>Garten</option>
              <option>Feste</option>
              <option>Reparatur</option>
              <option>Sonstiges</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Beschreibung</span>
            <textarea class="field-input min-h-28 resize-y py-3" required></textarea>
          </label>
        </div>

        <p v-if="submitted" class="mt-4 rounded-lg border bg-muted px-3 py-2 text-sm text-muted-foreground">
          Der API-Endpunkt für Einreichungen wird im nächsten Backend-Schnitt verbunden.
        </p>

        <button class="primary-button mt-5 w-full" type="submit">
          Einreichen
        </button>
      </form>

      <div class="rounded-lg border bg-surface p-5 shadow-soft">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-lg font-semibold">Historie</h2>
          <Clock3 class="h-5 w-5 text-muted-foreground" :stroke-width="1.8" />
        </div>
        <div class="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed bg-background px-5 text-center">
          <Check class="mb-4 h-8 w-8 text-primary" :stroke-width="1.8" />
          <p class="text-sm font-semibold">Noch keine Elternstunden</p>
          <p class="mt-2 max-w-[38ch] text-sm leading-6 text-muted-foreground">
            Nach dem Stammdaten-Sync erscheinen hier eingereichte und geprüfte Einträge.
          </p>
        </div>
      </div>
    </div>
  </section>
</template>
