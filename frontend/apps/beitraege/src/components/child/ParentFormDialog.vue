<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';
import { api } from '@/api';
import type { Child, CreateParentRequest, Parent } from '@/api/types';
import { Loader2, X, Plus, Link, Search } from 'lucide-vue-next';

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
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-xl font-semibold">
          {{ parentDialogMode === 'create' ? 'Elternteil anlegen' : 'Elternteil verknüpfen' }}
        </h2>
        <button @click="$emit('close')" class="p-1 hover:bg-gray-100 rounded">
          <X class="h-5 w-5" />
        </button>
      </div>

      <!-- Mode Tabs -->
      <div class="flex gap-2 mb-6 p-1 bg-gray-100 rounded-lg">
        <button
          @click="parentDialogMode = 'create'"
          :class="[
            'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors',
            parentDialogMode === 'create'
              ? 'bg-white text-primary shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          ]"
        >
          Neu anlegen
        </button>
        <button
          @click="parentDialogMode = 'link'"
          :class="[
            'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors',
            parentDialogMode === 'link'
              ? 'bg-white text-primary shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          ]"
        >
          Vorhandenen verknüpfen
        </button>
      </div>

      <!-- Create Form -->
      <form v-if="parentDialogMode === 'create'" @submit.prevent="handleCreateParent" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="parent-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname *</label>
            <input
              id="parent-firstName"
              v-model="parentForm.firstName"
              type="text"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="parent-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname *</label>
            <input
              id="parent-lastName"
              v-model="parentForm.lastName"
              type="text"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div>
          <label for="parent-email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
          <input
            id="parent-email"
            v-model="parentForm.email"
            type="email"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div>
          <label for="parent-phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
          <input
            id="parent-phone"
            v-model="parentForm.phone"
            type="tel"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div class="grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="parent-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              id="parent-street"
              v-model="parentForm.street"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="parent-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
            <input
              id="parent-streetNo"
              v-model="parentForm.streetNo"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div>
            <label for="parent-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
            <input
              id="parent-postalCode"
              v-model="parentForm.postalCode"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div class="col-span-2">
            <label for="parent-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
            <input
              id="parent-city"
              v-model="parentForm.city"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div v-if="parentError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ parentError }}</p>
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            @click="$emit('close')"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            :disabled="isCreatingParent"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isCreatingParent" class="h-4 w-4 animate-spin" />
            <Plus v-else class="h-4 w-4" />
            Anlegen & Verknüpfen
          </button>
        </div>
      </form>

      <!-- Link Form -->
      <div v-else class="space-y-4">
        <div>
          <label for="parent-search" class="block text-sm font-medium text-gray-700 mb-1">Elternteil suchen</label>
          <div class="relative">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              id="parent-search"
              v-model="searchQuery"
              type="text"
              placeholder="Name eingeben..."
              class="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
            <Loader2 v-if="isSearching" class="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-gray-400" />
          </div>
        </div>

        <!-- Search Results -->
        <div v-if="searchResults.length > 0" class="border rounded-lg divide-y max-h-60 overflow-y-auto">
          <button
            v-for="parent in searchResults"
            :key="parent.id"
            @click="selectParent(parent)"
            :class="[
              'w-full p-3 text-left hover:bg-gray-50 transition-colors',
              selectedParent?.id === parent.id ? 'bg-primary/5 border-l-2 border-l-primary' : ''
            ]"
          >
            <p class="font-medium">{{ parent.firstName }} {{ parent.lastName }}</p>
            <p v-if="parent.email" class="text-sm text-gray-500">{{ parent.email }}</p>
          </button>
        </div>

        <div v-else-if="searchQuery.length >= 2 && !isSearching" class="text-center py-6 text-gray-500 text-sm">
          Keine Eltern gefunden
        </div>

        <div v-else-if="searchQuery.length < 2" class="text-center py-6 text-gray-500 text-sm">
          Mindestens 2 Zeichen eingeben
        </div>

        <!-- Selected Parent Preview -->
        <div v-if="selectedParent" class="p-4 bg-primary/5 border border-primary/20 rounded-lg">
          <p class="text-sm text-gray-500 mb-1">Ausgewählt:</p>
          <p class="font-medium">{{ selectedParent.firstName }} {{ selectedParent.lastName }}</p>
          <p v-if="selectedParent.email" class="text-sm text-gray-600">{{ selectedParent.email }}</p>
        </div>

        <div v-if="parentError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ parentError }}</p>
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            @click="$emit('close')"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleLinkParent"
            :disabled="!selectedParent || isLinking"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isLinking" class="h-4 w-4 animate-spin" />
            <Link v-else class="h-4 w-4" />
            Verknüpfen
          </button>
        </div>
      </div>
    </div>
  </div></template>
