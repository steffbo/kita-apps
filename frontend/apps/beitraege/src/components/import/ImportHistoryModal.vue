<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import type { ImportBatch } from '@/api/types';
import ImportErrorList from '@/components/ImportErrorList.vue';
import { FileSpreadsheet, Loader2, XCircle, History } from 'lucide-vue-next';
import { formatDate, formatDateTime } from '@/utils/format';

const props = defineProps<{ expandBatchId?: string | null }>();
const emit = defineEmits<{
  close: [];
  // Newest batch after a reload, so the page can refresh its error banner
  loaded: [latest: ImportBatch | null];
}>();

const importHistory = ref<ImportBatch[]>([]);
const isLoadingHistory = ref(false);
const historyError = ref<string | null>(null);
const expandedBatchId = ref<string | null>(props.expandBatchId ?? null);

async function loadHistory(): Promise<void> {
  isLoadingHistory.value = true;
  historyError.value = null;
  try {
    const response = await api.getImportHistory(1, 50);
    importHistory.value = response.data;
    emit('loaded', response.data[0] ?? null);
  } catch (error) {
    historyError.value = error instanceof Error ? error.message : 'Import-Historie konnte nicht geladen werden';
  } finally {
    isLoadingHistory.value = false;
  }
}

function toggleBatchErrors(batchId: string): void {
  expandedBatchId.value = expandedBatchId.value === batchId ? null : batchId;
}

onMounted(loadHistory);
</script>

<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl max-w-4xl w-full mx-4 max-h-[85vh] overflow-hidden flex flex-col">
      <div class="p-4 border-b flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold">Import-Historie</h2>
          <p class="text-sm text-gray-600">Frühere CSV- und Sync-Importe</p>
        </div>
        <div class="flex items-center gap-3">
          <button @click="loadHistory" class="text-sm text-gray-600 hover:text-gray-900 underline">
            Aktualisieren
          </button>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XCircle class="h-5 w-5" />
          </button>
        </div>
      </div>

      <div class="overflow-y-auto p-4">
        <div v-if="historyError" class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {{ historyError }}
        </div>
        <div v-if="isLoadingHistory" class="flex items-center justify-center py-12">
          <Loader2 class="h-8 w-8 animate-spin text-primary" />
        </div>

        <div v-else-if="importHistory.length === 0" class="text-center py-12">
          <History class="h-12 w-12 text-gray-300 mx-auto mb-4" />
          <p class="text-gray-600">Noch keine Importe durchgeführt</p>
        </div>

        <div v-else class="rounded-xl border overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead class="bg-gray-50">
                <tr class="text-left text-sm text-gray-500">
                  <th class="px-4 py-3 font-medium">Datei</th>
                  <th class="px-4 py-3 font-medium">Zeitraum</th>
                  <th class="px-4 py-3 font-medium">Transaktionen</th>
                  <th class="px-4 py-3 font-medium">Zugeordnet</th>
                  <th class="px-4 py-3 font-medium">Fehler</th>
                  <th class="px-4 py-3 font-medium">Importiert am</th>
                  <th class="px-4 py-3 font-medium">Von</th>
                </tr>
              </thead>
              <tbody>
                <template v-for="batch in importHistory" :key="batch.id">
                <tr class="border-t hover:bg-gray-50">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <FileSpreadsheet class="h-4 w-4 text-gray-400" />
                      <span class="font-medium">{{ batch.fileName }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-gray-600 text-sm">
                    <span v-if="batch.dateFrom && batch.dateTo">
                      {{ formatDate(batch.dateFrom) }} - {{ formatDate(batch.dateTo) }}
                    </span>
                    <span v-else class="text-gray-400">-</span>
                  </td>
                  <td class="px-4 py-3">{{ batch.transactionCount }}</td>
                  <td class="px-4 py-3">
                    <span
                      :class="[
                        'px-2 py-0.5 rounded-full text-xs font-medium',
                        batch.matchedCount === batch.transactionCount
                          ? 'bg-green-100 text-green-700'
                          : batch.matchedCount > 0
                            ? 'bg-amber-100 text-amber-700'
                            : 'bg-gray-100 text-gray-700',
                      ]"
                    >
                      {{ batch.matchedCount }} / {{ batch.transactionCount }}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <button
                      v-if="batch.errorCount > 0"
                      @click="toggleBatchErrors(batch.id)"
                      class="px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700 hover:bg-red-200"
                    >
                      {{ batch.errorCount }} {{ expandedBatchId === batch.id ? '▲' : '▼' }}
                    </button>
                    <span v-else class="text-gray-400">–</span>
                  </td>
                  <td class="px-4 py-3 text-gray-600">
                    {{ formatDateTime(batch.importedAt) }}
                  </td>
                  <td class="px-4 py-3 text-gray-600">
                    {{ batch.importedByEmail || batch.importedBy }}
                  </td>
                </tr>
                <tr v-if="expandedBatchId === batch.id && batch.errorCount > 0" class="border-t">
                  <td colspan="7" class="px-4 py-3">
                    <ImportErrorList :errors="batch.errors ?? []" :total="batch.errorCount" />
                  </td>
                </tr>
                </template>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div></template>
