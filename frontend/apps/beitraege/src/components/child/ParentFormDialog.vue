<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';
import { api } from '@/api';
import type { Child, CreateParentRequest, Parent } from '@/api/types';
import { Loader2, Plus, Link } from 'lucide-vue-next';
import SearchInput from '@/components/SearchInput.vue';

const props = defineProps<{ child: Child; initialMode: 'create' | 'link' }>();
const emit = defineEmits<{ close: []; saved: [] }>();

const parentDialogMode = ref<'create' | 'link'>(props.initialMode);
const parentForm = ref<CreateParentRequest>({
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  street: '',
  streetNo: '',
  postalCode: '',
  city: '',
});
const isCreatingParent = ref(false);
const parentError = ref<string | null>(null);

// Link parent state
const searchQuery = ref('');
const searchResults = ref<Parent[]>([]);
const isSearching = ref(false);
const selectedParent = ref<Parent | null>(null);
const isLinking = ref(false);

async function handleCreateParent() {
  isCreatingParent.value = true;
  parentError.value = null;
  try {
    const newParent = await api.createParent(parentForm.value);
    await api.linkParent(props.child.id, newParent.id, props.child.parents?.length === 0);
    emit('saved');
  } catch (e) {
    parentError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreatingParent.value = false;
  }
}

async function searchParents() {
  if (!searchQuery.value || searchQuery.value.length < 2) {
    searchResults.value = [];
    return;
  }
  isSearching.value = true;
  try {
    const response = await api.getParents({ search: searchQuery.value, perPage: 10 });
    // Filter out parents already linked to this child
    const linkedIds = new Set(props.child.parents?.map(p => p.id) || []);
    searchResults.value = response.data.filter(p => !linkedIds.has(p.id));
  } catch (e) {
    parentError.value = e instanceof Error ? e.message : 'Fehler bei der Suche';
  } finally {
    isSearching.value = false;
  }
}

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout> | null = null;
watch(searchQuery, () => {
  if (searchTimeout) clearTimeout(searchTimeout);
  searchTimeout = setTimeout(searchParents, 300);
});

onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout);
});

function selectParent(parent: Parent) {
  selectedParent.value = parent;
}

async function handleLinkParent() {
  if (!selectedParent.value) return;
  isLinking.value = true;
  parentError.value = null;
  try {
    await api.linkParent(props.child.id, selectedParent.value.id, props.child.parents?.length === 0);
    emit('saved');
  } catch (e) {
    parentError.value = e instanceof Error ? e.message : 'Fehler beim Verknüpfen';
  } finally {
    isLinking.value = false;
  }
}
import AppSheet from '@/components/ui/AppSheet.vue';
</script>

<template>
  <AppSheet :title="parentDialogMode === 'create' ? 'Elternteil anlegen' : 'Elternteil verknüpfen'" hide-title @close="$emit('close')">

      <div class="flex items-center justify-between mb-6">
        <h2 class="text-base font-semibold">
          {{ parentDialogMode === 'create' ? 'Elternteil anlegen' : 'Elternteil verknüpfen' }}
        </h2>
      </div>

      <!-- Mode Tabs -->
      <div class="flex gap-2 mb-6 p-1 bg-muted rounded-lg">
        <button
          @click="parentDialogMode = 'create'"
          :class="[
            'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors',
            parentDialogMode === 'create'
              ? 'bg-card text-primary shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          ]"
        >
          Neu anlegen
        </button>
        <button
          @click="parentDialogMode = 'link'"
          :class="[
            'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors',
            parentDialogMode === 'link'
              ? 'bg-card text-primary shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          ]"
        >
          Vorhandenen verknüpfen
        </button>
      </div>

      <!-- Create Form -->
      <form id="parent-create-form" v-if="parentDialogMode === 'create'"
        @submit.prevent="handleCreateParent" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="parent-firstName" class="label">Vorname *</label>
            <input
              id="parent-firstName"
              v-model="parentForm.firstName"
              type="text"
              required
              class="input"
            />
          </div>
          <div>
            <label for="parent-lastName" class="label">Nachname *</label>
            <input
              id="parent-lastName"
              v-model="parentForm.lastName"
              type="text"
              required
              class="input"
            />
          </div>
        </div>

        <div>
          <label for="parent-email" class="label">E-Mail</label>
          <input
            id="parent-email"
            v-model="parentForm.email"
            type="email"
            class="input"
          />
        </div>

        <div>
          <label for="parent-phone" class="label">Telefon</label>
          <input
            id="parent-phone"
            v-model="parentForm.phone"
            type="tel"
            class="input"
          />
        </div>

        <div class="grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="parent-street" class="label">Straße</label>
            <input
              id="parent-street"
              v-model="parentForm.street"
              type="text"
              class="input"
            />
          </div>
          <div>
            <label for="parent-streetNo" class="label">Hausnr.</label>
            <input
              id="parent-streetNo"
              v-model="parentForm.streetNo"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div>
            <label for="parent-postalCode" class="label">PLZ</label>
            <input
              id="parent-postalCode"
              v-model="parentForm.postalCode"
              type="text"
              class="input"
            />
          </div>
          <div class="col-span-2">
            <label for="parent-city" class="label">Ort</label>
            <input
              id="parent-city"
              v-model="parentForm.city"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div v-if="parentError" class="note note-danger">
          <p class="text-sm text-destructive">{{ parentError }}</p>
        </div>

      </form>

      <!-- Link Form -->
      <div v-else class="space-y-4">
        <div>
          <label for="parent-search" class="label">Elternteil suchen</label>
          <SearchInput id="parent-search" v-model="searchQuery" placeholder="Name eingeben..."
            :loading="isSearching" />
        </div>

        <!-- Search Results -->
        <div v-if="searchResults.length > 0" class="border rounded-lg divide-y max-h-60 overflow-y-auto">
          <button
            v-for="parent in searchResults"
            :key="parent.id"
            @click="selectParent(parent)"
            :class="[
              'w-full p-3 text-left hover:bg-accent transition-colors',
              selectedParent?.id === parent.id ? 'bg-primary/5 border-l-2 border-l-primary' : ''
            ]"
          >
            <p class="font-medium">{{ parent.firstName }} {{ parent.lastName }}</p>
            <p v-if="parent.email" class="text-sm text-muted-foreground">{{ parent.email }}</p>
          </button>
        </div>

        <div v-else-if="searchQuery.length >= 2 && !isSearching" class="text-center py-6 text-muted-foreground text-sm">
          Keine Eltern gefunden
        </div>

        <div v-else-if="searchQuery.length < 2" class="text-center py-6 text-muted-foreground text-sm">
          Mindestens 2 Zeichen eingeben
        </div>

        <!-- Selected Parent Preview -->
        <div v-if="selectedParent" class="p-4 bg-primary/5 border border-primary/20 rounded-lg">
          <p class="text-sm text-muted-foreground mb-1">Ausgewählt:</p>
          <p class="font-medium">{{ selectedParent.firstName }} {{ selectedParent.lastName }}</p>
          <p v-if="selectedParent.email" class="text-sm text-muted-foreground">{{ selectedParent.email }}</p>
        </div>

        <div v-if="parentError" class="note note-danger">
          <p class="text-sm text-destructive">{{ parentError }}</p>
        </div>

      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="$emit('close')">Abbrechen</button>
        <button v-if="parentDialogMode === 'create'" type="submit" form="parent-create-form"
          class="btn btn-primary" :disabled="isCreatingParent">
          <Loader2 v-if="isCreatingParent" class="animate-spin" /><Plus v-else />Anlegen & Verknüpfen
        </button>
        <button v-else type="button" class="btn btn-primary"
          :disabled="!selectedParent || isLinking" @click="handleLinkParent">
          <Loader2 v-if="isLinking" class="animate-spin" /><Link v-else />Verknüpfen
        </button>
      </template>
  </AppSheet>
</template>
