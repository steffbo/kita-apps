<script setup lang="ts">
import { onMounted } from 'vue';
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
import { usePagedList } from '@/composables/usePagedList';

const {
  items: notes,
  total,
  isLoading,
  error,
  currentPage,
  pageSize,
  totalPages,
  offset,
  load: loadNotes,
  goToPage,
} = usePagedList<ChildNote>({ fetchPage: pagination => api.getNotes(pagination) });

onMounted(loadNotes);

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
        :total="total" :offset="offset" @go="goToPage" />
    </div>
  </div>
</template>
