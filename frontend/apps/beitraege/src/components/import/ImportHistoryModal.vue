<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import type { ImportBatch } from '@/api/types';
import ImportErrorList from '@/components/ImportErrorList.vue';
import { FileSpreadsheet, Loader2, History } from 'lucide-vue-next';
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
  <AppSheet title="Import-Historie" size="lg" @close="$emit('close')">
    <div class="mb-4 flex justify-end">
      <button @click="loadHistory" class="btn btn-ghost btn-sm">Aktualisieren</button>
    </div>
    <div>
        <div v-if="historyError" role="alert" class="note note-danger mb-4">
          {{ historyError }}
        </div>
        <div v-if="isLoadingHistory" class="flex items-center justify-center py-12">
          <Loader2 class="h-8 w-8 animate-spin text-primary" />
        </div>

        <div v-else-if="importHistory.length === 0" class="text-center py-12">
          <History class="h-12 w-12 text-muted-foreground/60 mx-auto mb-4" />
          <p class="text-muted-foreground">Noch keine Importe durchgeführt</p>
        </div>

        <div v-else class="rounded-xl border overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead class="bg-muted">
                <tr class="text-left text-sm text-muted-foreground">
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
                <tr class="border-t hover:bg-accent">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <FileSpreadsheet class="h-4 w-4 text-muted-foreground" />
                      <span class="font-medium">{{ batch.fileName }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-muted-foreground text-sm">
                    <span v-if="batch.dateFrom && batch.dateTo">
                      {{ formatDate(batch.dateFrom) }} - {{ formatDate(batch.dateTo) }}
                    </span>
                    <span v-else class="text-muted-foreground">-</span>
                  </td>
                  <td class="px-4 py-3">{{ batch.transactionCount }}</td>
                  <td class="px-4 py-3">
                    <span
                      :class="[
                        'px-2 py-0.5 rounded-full text-xs font-medium',
                        batch.matchedCount === batch.transactionCount
                          ? 'bg-success-soft text-success'
                          : batch.matchedCount > 0
                            ? 'bg-warning-soft text-warning'
                            : 'bg-muted text-foreground',
                      ]"
                    >
                      {{ batch.matchedCount }} / {{ batch.transactionCount }}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <button
                      v-if="batch.errorCount > 0"
                      @click="toggleBatchErrors(batch.id)"
                      class="px-2 py-0.5 rounded-full text-xs font-medium bg-destructive-soft text-destructive hover:bg-destructive-soft"
                    >
                      {{ batch.errorCount }} {{ expandedBatchId === batch.id ? '▲' : '▼' }}
                    </button>
                    <span v-else class="text-muted-foreground">–</span>
                  </td>
                  <td class="px-4 py-3 text-muted-foreground">
                    {{ formatDateTime(batch.importedAt) }}
                  </td>
                  <td class="px-4 py-3 text-muted-foreground">
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
</AppSheet>
</template>
