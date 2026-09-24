<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { GenerateFeeRequest } from '@/api/types';
import { Loader2, Plus, Calendar } from 'lucide-vue-next';
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
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <Calendar class="h-6 w-6 text-primary" />
        </div>
        <div>
          <h2 class="text-xl font-semibold">Beiträge generieren</h2>
          <p class="text-sm text-gray-600">Erstellt fehlende Beiträge für alle aktiven Kinder</p>
        </div>
      </div>

      <div class="space-y-4">
        <div>
          <label for="generate-year" class="block text-sm font-medium text-gray-700 mb-1">Jahr</label>
          <select
            id="generate-year"
            v-model="generateForm.year"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
          </select>
        </div>

        <div>
          <label for="generate-month" class="block text-sm font-medium text-gray-700 mb-1">Monat</label>
          <select
            id="generate-month"
            v-model="generateForm.month"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option :value="undefined">Nur Jahresbeitrag (Vereinsbeitrag)</option>
            <option v-for="month in months" :key="month.value" :value="month.value">
              {{ month.label }} (Essensgeld + ggf. Platzgeld)
            </option>
          </select>
        </div>

        <div v-if="generateResult" class="p-4 bg-green-50 border border-green-200 rounded-lg">
          <p class="text-green-700 font-medium">Erfolgreich generiert!</p>
          <p class="text-sm text-green-600 mt-1">
            {{ generateResult.created }} Beiträge erstellt, {{ generateResult.skipped }} übersprungen
          </p>
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            @click="$emit('close')"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Schließen
          </button>
          <button
            @click="handleGenerate"
            :disabled="isGenerating"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
            <Plus v-else class="h-4 w-4" />
            Generieren
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
