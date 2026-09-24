<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { api } from '@/api';
import type { ChildNote } from '@/api/types';
import { Edit, Trash2, Loader2, Plus, X, Check, NotebookPen, ChevronLeft, ChevronRight } from 'lucide-vue-next';
import { formatDateTime } from '@/utils/format';

const props = defineProps<{ childId: string }>();

const notes = ref<ChildNote[]>([]);
const notesTotal = ref(0);
const notesIsLoading = ref(false);
const notesError = ref<string | null>(null);
const notesPage = ref(1);
const notesPageSize = 5;
const notesTotalPages = computed(() => Math.max(1, Math.ceil(notesTotal.value / notesPageSize)));

// Note dialog state
const showNoteDialog = ref(false);
const noteDialogMode = ref<'create' | 'edit'>('create');
const editingNote = ref<ChildNote | null>(null);
const noteForm = ref({ text: '' });
const isSavingNote = ref(false);
const noteError = ref<string | null>(null);

// Note delete dialog state
const showNoteDeleteDialog = ref(false);
const noteToDelete = ref<ChildNote | null>(null);
const isDeletingNote = ref(false);

watch(() => props.childId, () => {
  if (notesPage.value === 1) loadNotes();
  else notesPage.value = 1;
});

watch(notesPage, () => {
  loadNotes();
});

let loadNotesSeq = 0;
async function loadNotes() {
  const seq = ++loadNotesSeq;
  notesIsLoading.value = true;
  notesError.value = null;
  try {
    const response = await api.getChildNotes(props.childId, {
      page: notesPage.value,
      perPage: notesPageSize,
    });
    if (seq !== loadNotesSeq) return;
    notes.value = response.data;
    notesTotal.value = response.total;

    // If the page ran empty (e.g. after deleting the last note of a page),
    // fall back to the last valid page
    if (response.data.length === 0 && notesPage.value > 1 && response.total > 0) {
      notesPage.value = Math.max(1, Math.ceil(response.total / notesPageSize));
      return;
    }
  } catch (e) {
    if (seq !== loadNotesSeq) return;
    notesError.value = e instanceof Error ? e.message : 'Fehler beim Laden der Notizen';
  } finally {
    if (seq === loadNotesSeq) notesIsLoading.value = false;
  }
}

function openNoteCreateDialog() {
  noteDialogMode.value = 'create';
  editingNote.value = null;
  noteForm.value = { text: '' };
  noteError.value = null;
  showNoteDialog.value = true;
}

function openNoteEditDialog(note: ChildNote) {
  noteDialogMode.value = 'edit';
  editingNote.value = note;
  noteForm.value = { text: note.text };
  noteError.value = null;
  showNoteDialog.value = true;
}

async function handleSaveNote() {
  isSavingNote.value = true;
  noteError.value = null;
  try {
    if (noteDialogMode.value === 'create') {
      await api.createChildNote(props.childId, { text: noteForm.value.text });
    } else if (editingNote.value) {
      await api.updateChildNote(props.childId, editingNote.value.id, { text: noteForm.value.text });
    }
    showNoteDialog.value = false;
    editingNote.value = null;
    await loadNotes();
  } catch (e) {
    noteError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isSavingNote.value = false;
  }
}

async function handleDeleteNote() {
  if (!noteToDelete.value) return;
  isDeletingNote.value = true;
  try {
    await api.deleteChildNote(props.childId, noteToDelete.value.id);
    showNoteDeleteDialog.value = false;
    noteToDelete.value = null;
    await loadNotes();
  } catch (e) {
    notesError.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showNoteDeleteDialog.value = false;
  } finally {
    isDeletingNote.value = false;
  }
}

// Shows a "Bearbeitet" marker when the update timestamp drifted from creation.
function isNoteEdited(note: ChildNote): boolean {
  return new Date(note.updatedAt).getTime() - new Date(note.createdAt).getTime() > 1000;
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return;
  if (showNoteDeleteDialog.value) {
    showNoteDeleteDialog.value = false;
  } else if (showNoteDialog.value) {
    showNoteDialog.value = false;
  }
}

onMounted(() => {
  loadNotes();
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<template>
  <div class="bg-white rounded-xl border p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <NotebookPen class="h-5 w-5 text-primary" />
        <h2 class="text-lg font-semibold">Notizen</h2>
      </div>
      <button
        @click="openNoteCreateDialog"
        class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium bg-primary text-white hover:bg-primary/90 rounded-md transition-colors"
      >
        <Plus class="h-3 w-3" />
        Notiz
      </button>
    </div>

    <div v-if="notesIsLoading" class="flex items-center gap-2 text-sm text-gray-500 py-4">
      <Loader2 class="h-4 w-4 animate-spin" />
      Lade Notizen...
    </div>

    <div v-else-if="notesError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
      <p class="text-sm text-red-600">{{ notesError }}</p>
    </div>

    <div v-else-if="notes.length === 0" class="text-center py-6 text-gray-500 text-sm">
      Keine Notizen vorhanden
    </div>

    <div v-else class="space-y-2">
      <div
        v-for="note in notes"
        :key="note.id"
        class="p-3 bg-gray-50 border border-gray-200 rounded-lg"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm text-gray-700 whitespace-pre-wrap break-words">{{ note.text }}</p>
            <p class="text-xs text-gray-400 mt-1">
              {{ formatDateTime(note.createdAt) }}
              <span v-if="isNoteEdited(note)"> · Bearbeitet: {{ formatDateTime(note.updatedAt) }}</span>
            </p>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button
              @click="openNoteEditDialog(note)"
              class="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-4 w-4" />
            </button>
            <button
              @click="noteToDelete = note; showNoteDeleteDialog = true"
              class="p-1.5 text-red-500 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors"
              title="Löschen"
            >
              <Trash2 class="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      <div v-if="notesTotalPages > 1" class="flex items-center justify-end gap-2 pt-2">
        <span class="text-xs text-gray-500">Seite {{ notesPage }} von {{ notesTotalPages }}</span>
        <button
          @click="notesPage--"
          :disabled="notesPage <= 1"
          class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <ChevronLeft class="h-4 w-4" />
        </button>
        <button
          @click="notesPage++"
          :disabled="notesPage >= notesTotalPages"
          class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <ChevronRight class="h-4 w-4" />
        </button>
      </div>
    </div>
  </div>

  <div
    v-if="showNoteDialog"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="showNoteDialog = false"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-xl font-semibold">
          {{ noteDialogMode === 'create' ? 'Notiz erstellen' : 'Notiz bearbeiten' }}
        </h2>
        <button @click="showNoteDialog = false" class="p-1 hover:bg-gray-100 rounded">
          <X class="h-5 w-5" />
        </button>
      </div>

      <form @submit.prevent="handleSaveNote" class="space-y-4">
        <div>
          <label for="note-text" class="block text-sm font-medium text-gray-700 mb-1">Text *</label>
          <textarea
            id="note-text"
            v-model="noteForm.text"
            required
            rows="4"
            placeholder="Notiz eingeben..."
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none resize-y"
          ></textarea>
        </div>

        <div v-if="noteError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ noteError }}</p>
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            @click="showNoteDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            :disabled="isSavingNote || !noteForm.text.trim()"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isSavingNote" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
            Speichern
          </button>
        </div>
      </form>
    </div>
  </div>

  <!-- Note Delete Confirmation Dialog -->
  <div
    v-if="showNoteDeleteDialog && noteToDelete"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="showNoteDeleteDialog = false"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
      <div class="flex items-center gap-3 mb-4">
        <div class="p-2 bg-red-100 rounded-lg">
          <Trash2 class="h-6 w-6 text-red-600" />
        </div>
        <h2 class="text-xl font-semibold">Notiz löschen?</h2>
      </div>

      <p class="text-gray-600 mb-6">
        Möchtest du diese Notiz wirklich löschen?
        Diese Aktion kann nicht rückgängig gemacht werden.
      </p>

      <div class="flex justify-end gap-3">
        <button
          @click="showNoteDeleteDialog = false"
          class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
        >
          Abbrechen
        </button>
        <button
          @click="handleDeleteNote"
          :disabled="isDeletingNote"
          class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
        >
          <Loader2 v-if="isDeletingNote" class="h-4 w-4 animate-spin" />
          <Trash2 v-else class="h-4 w-4" />
          Löschen
        </button>
      </div>
    </div>
  </div></template>
