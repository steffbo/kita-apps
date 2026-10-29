<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';

import { ref } from 'vue';
import { api } from '@/api';
import type { GenerateFeeRequest } from '@/api/types';
import { Info, Loader2, Plus } from 'lucide-vue-next';
import { MONTH_OPTIONS as months } from '@/utils/format';

const emit = defineEmits<{ close: []; generated: [] }>();

const currentYear = new Date().getFullYear();
const years = [currentYear - 1, currentYear, currentYear + 1];

const generateForm = ref<GenerateFeeRequest>({
  year: currentYear,
  month: new Date().getMonth() + 1,
});
const isGenerating = ref(false);
const generateResult = ref<{ created: number; skipped: number } | null>(null);
const error = ref<string | null>(null);

async function handleGenerate() {
  isGenerating.value = true;
  generateResult.value = null;
  error.value = null;
  try {
    generateResult.value = await api.generateFees(generateForm.value);
    emit('generated');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Generieren';
  } finally {
    isGenerating.value = false;
  }
}
</script>

<template>
  <AppSheet title="Beiträge generieren" @close="$emit('close')">
    <p class="mb-4 text-sm text-muted-foreground">Erstellt fehlende Beiträge für alle aktiven Kinder</p>
    <form id="generate-fees" class="space-y-4" @submit.prevent="handleGenerate">
      <div>
        <label for="generate-year" class="label">Jahr</label>
        <select id="generate-year" v-model="generateForm.year" class="input">
          <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
        </select>
      </div>
      <div>
        <label for="generate-month" class="label">Monat</label>
        <select id="generate-month" v-model="generateForm.month" class="input">
          <option :value="undefined">Nur Jahresbeitrag (Vereinsbeitrag)</option>
          <option v-for="month in months" :key="month.value" :value="month.value">
            {{ month.label }} (Essensgeld + ggf. Platzgeld)
          </option>
        </select>
      </div>
      <div v-if="generateResult" class="note note-success">
        <Info />
        <div class="min-w-0 flex-1">
          <p class="text-success font-medium">Erfolgreich generiert!</p>
          <p class="text-sm text-success mt-1">
            {{ generateResult.created }} Beiträge erstellt, {{ generateResult.skipped }} übersprungen
          </p>
        </div>
      </div>
    </form>
    <template #footer>
      <button type="button" @click="$emit('close')" class="btn btn-secondary">Abbrechen</button>
      <button type="submit" form="generate-fees" :disabled="isGenerating" class="btn btn-primary">
        <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
        <Plus v-else class="h-4 w-4" />
        Generieren
      </button>
    </template>
  </AppSheet>
</template>
