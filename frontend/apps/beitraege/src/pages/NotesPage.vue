<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { api } from '@/api';
import type { ChildNote } from '@/api/types';
import {
  Loader2,
  ChevronLeft,
  ChevronRight,
  User,
} from 'lucide-vue-next';

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
  loadNotes();
}

onMounted(() => {
  loadNotes();
});

// Helpers
function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString('de-DE');
}

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

// Pagination display helpers
const visiblePages = computed(() => {
  const pages: (number | '...')[] = [];
  const total = totalPages.value;
  const current = currentPage.value;

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i);
  } else {
    pages.push(1);
    if (current > 3) pages.push('...');

    const start = Math.max(2, current - 1);
    const end = Math.min(total - 1, current + 1);

    for (let i = start; i <= end; i++) pages.push(i);

    if (current < total - 2) pages.push('...');
    pages.push(total);
  }

  return pages;
});
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Notizen</h1>
      <p class="text-gray-600 mt-1">{{ total }} Notizen zu Kindern</p>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadNotes" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Notes table -->
    <div v-else class="bg-white rounded-xl border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th class="px-4 py-3 font-medium">Notiz</th>
              <th class="px-4 py-3 font-medium">
                <span class="flex items-center gap-1">
                  <User class="h-4 w-4" />
                  Kind
                </span>
              </th>
              <th class="px-4 py-3 font-medium">Erstellt</th>
              <th class="hidden md:table-cell px-4 py-3 font-medium">Bearbeitet</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="note in notes" :key="note.id" class="border-t hover:bg-gray-50 transition-colors">
              <td class="px-4 py-3">
                <p class="text-gray-700 max-w-md">{{ truncateText(note.text) }}</p>
              </td>
              <td class="px-4 py-3">
                <router-link
                  :to="`/kinder/${note.childId}`"
                  class="text-primary hover:underline whitespace-nowrap"
                >
                  {{ childName(note) }}
                </router-link>
              </td>
              <td class="px-4 py-3 text-gray-600 whitespace-nowrap">
                {{ formatDateTime(note.createdAt) }}
              </td>
              <td class="hidden md:table-cell px-4 py-3 text-gray-500 whitespace-nowrap">
                <template v-if="isEdited(note)">{{ formatDateTime(note.updatedAt) }}</template>
                <span v-else class="text-gray-300">-</span>
              </td>
            </tr>
            <tr v-if="notes.length === 0">
              <td colspan="4" class="px-4 py-8 text-center text-gray-500">
                Keine Notizen vorhanden
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="flex flex-col sm:flex-row items-center justify-between gap-3 px-4 py-3 border-t bg-gray-50">
        <div class="flex items-center gap-4">
          <span class="text-sm text-gray-600">
            {{ offset + 1 }}-{{ Math.min(offset + pageSize, total) }} von {{ total }}
          </span>
          <select
            v-model="pageSize"
            @change="handlePageSizeChange"
            class="text-sm border border-gray-300 rounded px-2 py-1 focus:ring-primary focus:border-primary"
          >
            <option v-for="size in pageSizeOptions" :key="size" :value="size">
              {{ size }} pro Seite
            </option>
          </select>
        </div>
        <div class="flex items-center gap-1">
          <button
            @click="goToPage(currentPage - 1)"
            :disabled="currentPage === 1"
            class="p-1.5 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="h-4 w-4" />
          </button>
          <template v-for="page in visiblePages" :key="page">
            <span v-if="page === '...'" class="px-2 text-gray-400">...</span>
            <button
              v-else
              @click="goToPage(page)"
              :class="[
                'px-3 py-1 rounded text-sm',
                page === currentPage
                  ? 'bg-primary text-white'
                  : 'hover:bg-gray-200',
              ]"
            >
              {{ page }}
            </button>
          </template>
          <button
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === totalPages"
            class="p-1.5 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
