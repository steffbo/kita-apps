<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
import { api } from '@/api';
import type { ChildNote } from '@/api/types';
import { Edit, Trash2, Loader2, Plus, Check, NotebookPen, } from 'lucide-vue-next';
import { formatDateTime } from '@/utils/format';
import AppSheet from '@/components/ui/AppSheet.vue';
import AppPagination from '@/components/ui/AppPagination.vue';
import EmptyState from '@/components/ui/EmptyState.vue';

const props = defineProps<{ childId: string }>();

const notes = ref<ChildNote[]>([]);
const notesTotal = ref(0);
const notesIsLoading = ref(false);
const notesError = ref<string | null>(null);
const notesPage = ref(1);
const notesPageSize = ref(5);
const notesTotalPages = computed(() => Math.max(1, Math.ceil(notesTotal.value / notesPageSize.value)));

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
      perPage: notesPageSize.value,
    });
    if (seq !== loadNotesSeq) return;
    notes.value = response.data;
    notesTotal.value = response.total;

    // If the page ran empty (e.g. after deleting the last note of a page),
    // fall back to the last valid page
    if (response.data.length === 0 && notesPage.value > 1 && response.total > 0) {
      notesPage.value = Math.max(1, Math.ceil(response.total / notesPageSize.value));
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

onMounted(() => {
  loadNotes();
});
</script>

<template>
  <div class="card p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <NotebookPen class="h-5 w-5 text-primary" />
        <h2 class="text-base font-semibold">Notizen</h2>
      </div>
      <button data-write
        @click="openNoteCreateDialog"
        class="btn btn-secondary btn-sm"
      >
        <Plus class="h-3 w-3" />
        Notiz
      </button>
    </div>

    <div v-if="notesIsLoading" class="flex items-center gap-2 text-sm text-muted-foreground py-4">
      <Loader2 class="h-4 w-4 animate-spin" />
      Lade Notizen...
    </div>

    <div v-else-if="notesError" class="note note-danger">
      <p class="text-sm text-destructive">{{ notesError }}</p>
    </div>

    <EmptyState v-else-if="notes.length === 0" :icon="NotebookPen" text="Keine Notizen vorhanden" />

    <div v-else class="space-y-2">
      <div
        v-for="note in notes"
        :key="note.id"
        class="p-3 bg-muted border border-border rounded-lg"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm text-foreground whitespace-pre-wrap break-words">{{ note.text }}</p>
            <p class="text-xs text-muted-foreground mt-1">
              {{ formatDateTime(note.createdAt) }}
              <span v-if="isNoteEdited(note)"> · Bearbeitet: {{ formatDateTime(note.updatedAt) }}</span>
            </p>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button data-write
              @click="openNoteEditDialog(note)"
              class="p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-4 w-4" />
            </button>
            <button data-write
              @click="noteToDelete = note; showNoteDeleteDialog = true"
              class="p-1.5 text-destructive hover:text-destructive hover:bg-destructive-soft rounded-lg transition-colors"
              title="Löschen"
            >
              <Trash2 class="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      <AppPagination v-if="notesTotalPages > 1" v-model:page-size="notesPageSize"
        :current-page="notesPage" :total-pages="notesTotalPages" :total="notesTotal"
        :offset="(notesPage - 1) * notesPageSize" :page-size-options="[5]"
        @go="notesPage = $event" />
    </div>
  </div>

  <AppSheet v-if="showNoteDialog"
    :title="noteDialogMode === 'create' ? 'Notiz erstellen' : 'Notiz bearbeiten'"
    hide-title @close="showNoteDialog = false">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-base font-semibold">
          {{ noteDialogMode === 'create' ? 'Notiz erstellen' : 'Notiz bearbeiten' }}
        </h2>
      </div>

      <form id="child-note-form" @submit.prevent="handleSaveNote" class="space-y-4">
        <div>
          <label for="note-text" class="label">Text *</label>
          <textarea
            id="note-text"
            v-model="noteForm.text"
            required
            rows="4"
            placeholder="Notiz eingeben..."
            class="input resize-y"
          ></textarea>
        </div>

        <div v-if="noteError" class="note note-danger">
          <p class="text-sm text-destructive">{{ noteError }}</p>
        </div>

      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showNoteDialog = false">Abbrechen</button>
        <button type="submit" form="child-note-form" class="btn btn-primary"
          :disabled="isSavingNote || !noteForm.text.trim()">
          <Loader2 v-if="isSavingNote" class="animate-spin" /><Check v-else />Speichern
        </button>
      </template>
  </AppSheet>

  <!-- Note Delete Confirmation Dialog -->
  <AppSheet v-if="showNoteDeleteDialog && noteToDelete" title="Notiz löschen?" hide-title
    @close="showNoteDeleteDialog = false">
      <div class="flex items-center gap-3 mb-4">
        <div class="p-2 bg-destructive-soft rounded-lg">
          <Trash2 class="h-6 w-6 text-destructive" />
        </div>
        <h2 class="text-base font-semibold">Notiz löschen?</h2>
      </div>

      <p class="text-muted-foreground mb-6">
        Möchtest du diese Notiz wirklich löschen?
        Diese Aktion kann nicht rückgängig gemacht werden.
      </p>

      <template #footer>
        <button
          @click="showNoteDeleteDialog = false"
          class="btn btn-secondary"
        >
          Abbrechen
        </button>
        <button
          @click="handleDeleteNote"
          :disabled="isDeletingNote"
          class="btn btn-danger"
        >
          <Loader2 v-if="isDeletingNote" class="h-4 w-4 animate-spin" />
          <Trash2 v-else class="h-4 w-4" />
          Löschen
        </button>
      </template>
  </AppSheet>
</template>
