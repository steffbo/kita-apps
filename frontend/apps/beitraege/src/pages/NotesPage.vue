<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { api } from '@/api';
import type { ChildNote } from '@/api/types';
import {
  AlertTriangle,
  Loader2,
  User,
} from 'lucide-vue-next';
import { formatDateTime } from '@/utils/format';
import AppPagination from '@/components/ui/AppPagination.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';

// Data
const notes = ref<ChildNote[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Pagination
const currentPage = ref(1);
const pageSize = ref(25);
const pageSizeOptions = [10, 25, 50, 100];

const totalPages = computed(() => Math.ceil(total.value / pageSize.value));
const offset = computed(() => (currentPage.value - 1) * pageSize.value);

let loadNotesSeq = 0;
async function loadNotes() {
  const seq = ++loadNotesSeq;
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getNotes({
      page: currentPage.value,
      perPage: pageSize.value,
    });
    if (seq !== loadNotesSeq) return; // a newer request superseded this one
    notes.value = response.data;
    total.value = response.total;

    // If the current page ran empty (e.g. after deletes), fall back to the last valid page
    if (response.data.length === 0 && currentPage.value > 1 && response.total > 0) {
      currentPage.value = Math.max(1, Math.ceil(response.total / pageSize.value));
      return;
    }
  } catch (e) {
    if (seq !== loadNotesSeq) return;
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    if (seq === loadNotesSeq) isLoading.value = false;
  }
}

// Reload when pagination changes
watch([currentPage, pageSize], () => {
  loadNotes();
});

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
  }
}

// Change of page size restarts at page 1
function handlePageSizeChange() {
  currentPage.value = 1;
}

onMounted(() => {
  loadNotes();
});

// Helpers

function truncateText(text: string, length = 120): string {
  if (text.length <= length) return text;
  return `${text.slice(0, length).trimEnd()}…`;
}

function childName(note: ChildNote): string {
  return note.childName ?? 'Unbekanntes Kind';
}

// Shows an "Bearbeitet" marker when the update timestamp drifted from creation.
function isEdited(note: ChildNote): boolean {
  return new Date(note.updatedAt).getTime() - new Date(note.createdAt).getTime() > 1000;
}

</script>

<template>
  <div>
    <PageHeader title="Notizen" />
    <div class="mb-4 flex gap-2">
      <span class="chip pointer-events-none border-transparent bg-transparent px-1 text-muted-foreground">
        {{ total }} {{ total === 1 ? 'Notiz' : 'Notizen' }}
      </span>
    </div>

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>
    <div v-else-if="error" class="note note-danger">
      <AlertTriangle />
      <div>
        <p>{{ error }}</p>
      <button class="btn btn-secondary btn-sm" @click="loadNotes">Erneut versuchen</button>
      </div>
    </div>
    <div v-else class="card overflow-hidden">
      <ul class="divide-y sm:hidden">
        <li v-for="note in notes" :key="note.id" class="p-4">
          <p class="text-foreground">{{ truncateText(note.text) }}</p>
          <div class="mt-2 flex flex-wrap gap-x-2 text-[13px] text-muted-foreground">
            <router-link :to="`/kinder/${note.childId}`" class="text-primary hover:underline">
              {{ childName(note) }}
            </router-link>
            <span>{{ formatDateTime(note.createdAt) }}</span>
            <span v-if="isEdited(note)">Bearbeitet: {{ formatDateTime(note.updatedAt) }}</span>
          </div>
        </li>
        <li v-if="notes.length === 0"><EmptyState :icon="User" text="Keine Notizen vorhanden" /></li>
      </ul>
      <div class="hidden overflow-x-auto sm:block">
        <table class="table">
          <thead>
            <tr>
              <th>Notiz</th>
              <th><span class="flex items-center gap-1"><User />Kind</span></th>
              <th>Erstellt</th>
              <th class="hidden md:table-cell">Bearbeitet</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="note in notes" :key="note.id">
              <td><p class="max-w-md">{{ truncateText(note.text) }}</p></td>
              <td>
                <router-link :to="`/kinder/${note.childId}`" class="text-primary hover:underline whitespace-nowrap">
                  {{ childName(note) }}
                </router-link>
              </td>
              <td class="text-muted-foreground whitespace-nowrap">{{ formatDateTime(note.createdAt) }}</td>
              <td class="hidden md:table-cell text-muted-foreground whitespace-nowrap">
                <template v-if="isEdited(note)">{{ formatDateTime(note.updatedAt) }}</template>
                <span v-else>-</span>
              </td>
            </tr>
            <tr v-if="notes.length === 0"><td colspan="4"><EmptyState :icon="User" text="Keine Notizen vorhanden" /></td></tr>
          </tbody>
        </table>
      </div>
      <AppPagination v-model:page-size="pageSize" :current-page="currentPage" :total-pages="totalPages"
        :total="total" :offset="offset" :page-size-options="pageSizeOptions" @go="goToPage" @update:page-size="handlePageSizeChange" />
    </div>
  </div>
</template>
