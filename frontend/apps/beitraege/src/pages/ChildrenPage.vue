<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type { Child, CreateChildRequest } from '@/api/types';
import {
  Plus,
  Search,
  Loader2,
  User,
  Calendar,
  Hash,
  X,
  Check,
} from 'lucide-vue-next';

const router = useRouter();

const children = ref<Child[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);
const searchQuery = ref('');
const showInactive = ref(false);
const showCreateDialog = ref(false);

// Create form
const createForm = ref<CreateChildRequest>({
  memberNumber: '',
  firstName: '',
  lastName: '',
  birthDate: '',
  entryDate: '',
});
const isCreating = ref(false);
const createError = ref<string | null>(null);

async function loadChildren() {
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getChildren({
      activeOnly: !showInactive.value,
      search: searchQuery.value || undefined,
      limit: 100,
    });
    children.value = response.data;
    total.value = response.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadChildren);

// ESC key handler to close modal
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && showCreateDialog.value) {
    showCreateDialog.value = false;
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function calculateAge(birthDate: string): number {
  const birth = new Date(birthDate);
  const today = new Date();
  let age = today.getFullYear() - birth.getFullYear();
  const m = today.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birth.getDate())) {
    age--;
  }
  return age;
}

function isUnderThree(birthDate: string): boolean {
  return calculateAge(birthDate) < 3;
}

function goToChild(id: string) {
  router.push(`/kinder/${id}`);
}

async function handleCreate() {
  isCreating.value = true;
  createError.value = null;
  try {
    const child = await api.createChild(createForm.value);
    children.value.unshift(child);
    total.value++;
    showCreateDialog.value = false;
    createForm.value = {
      memberNumber: '',
      firstName: '',
      lastName: '',
      birthDate: '',
      entryDate: '',
    };
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreating.value = false;
  }
}

const filteredChildren = computed(() => children.value);
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Kinder</h1>
        <p class="text-gray-600 mt-1">{{ total }} Kinder registriert</p>
      </div>
      <button
        @click="showCreateDialog = true"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Kind hinzufügen
      </button>
    </div>

    <!-- Filters -->
    <div class="flex flex-col sm:flex-row gap-4 mb-6">
      <div class="relative flex-1">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
        <input
          v-model="searchQuery"
          @input="loadChildren"
          type="text"
          placeholder="Suchen nach Name oder Mitgliedsnummer..."
          class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        />
      </div>
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          v-model="showInactive"
          @change="loadChildren"
          type="checkbox"
          class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
        />
        <span class="text-sm text-gray-700">Inaktive anzeigen</span>
      </label>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadChildren" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Children list -->
    <div v-else class="bg-white rounded-xl border overflow-hidden">
      <table class="w-full">
        <thead class="bg-gray-50">
          <tr class="text-left text-sm text-gray-500">
            <th class="px-4 py-3 font-medium">
              <div class="flex items-center gap-2">
                <Hash class="h-4 w-4" />
                Nr.
              </div>
            </th>
            <th class="px-4 py-3 font-medium">
              <div class="flex items-center gap-2">
                <User class="h-4 w-4" />
                Name
              </div>
            </th>
            <th class="px-4 py-3 font-medium">
              <div class="flex items-center gap-2">
                <Calendar class="h-4 w-4" />
                Geburtsdatum
              </div>
            </th>
            <th class="px-4 py-3 font-medium">Alter</th>
            <th class="px-4 py-3 font-medium">U3</th>
            <th class="px-4 py-3 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="child in filteredChildren"
            :key="child.id"
            @click="goToChild(child.id)"
            class="border-t hover:bg-gray-50 cursor-pointer transition-colors"
          >
            <td class="px-4 py-3 font-mono text-sm">{{ child.memberNumber }}</td>
            <td class="px-4 py-3 font-medium">{{ child.firstName }} {{ child.lastName }}</td>
            <td class="px-4 py-3 text-gray-600">{{ formatDate(child.birthDate) }}</td>
            <td class="px-4 py-3">{{ calculateAge(child.birthDate) }} Jahre</td>
            <td class="px-4 py-3">
              <span
                v-if="isUnderThree(child.birthDate)"
                class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700"
              >
                U3
              </span>
              <span v-else class="text-gray-400">-</span>
            </td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                  child.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600',
                ]"
              >
                {{ child.isActive ? 'Aktiv' : 'Inaktiv' }}
              </span>
            </td>
          </tr>
          <tr v-if="filteredChildren.length === 0">
            <td colspan="6" class="px-4 py-8 text-center text-gray-500">
              Keine Kinder gefunden
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Dialog -->
    <div
      v-if="showCreateDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showCreateDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Kind hinzufügen</h2>
          <button @click="showCreateDialog = false" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <form @submit.prevent="handleCreate" class="space-y-4">
          <div>
            <label for="memberNumber" class="block text-sm font-medium text-gray-700 mb-1">Mitgliedsnummer *</label>
            <input
              id="memberNumber"
              v-model="createForm.memberNumber"
              required
              type="text"
              placeholder="z.B. 11072"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname *</label>
              <input
                id="firstName"
                v-model="createForm.firstName"
                required
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname *</label>
              <input
                id="lastName"
                v-model="createForm.lastName"
                required
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum *</label>
              <input
                id="birthDate"
                v-model="createForm.birthDate"
                required
                type="date"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="entryDate" class="block text-sm font-medium text-gray-700 mb-1">Eintrittsdatum *</label>
              <input
                id="entryDate"
                v-model="createForm.entryDate"
                required
                type="date"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div v-if="createError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ createError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showCreateDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isCreating"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isCreating" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
