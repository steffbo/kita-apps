<script setup lang="ts">
import { ref, onUnmounted } from 'vue';
import { api } from '@/api';
import type { Parent, UpdateParentRequest } from '@/api/types';
import { Edit, Loader2, User, X, Check, Copy } from 'lucide-vue-next';
import { formatDate, formatDateForInput } from '@/utils/format';

const props = defineProps<{ parent: Parent }>();
const emit = defineEmits<{ close: []; saved: [] }>();

// Local copy so the modal shows the saved values right away.
const current = ref<Parent>(props.parent);
const isEditingParent = ref(false);
const parentEditForm = ref<UpdateParentRequest>({});
const isSavingParent = ref(false);
const parentDetailError = ref<string | null>(null);
const isParentEmailCopied = ref(false);
let parentEmailCopyResetTimer: ReturnType<typeof setTimeout> | null = null;

onUnmounted(() => {
  if (parentEmailCopyResetTimer) {
    clearTimeout(parentEmailCopyResetTimer);
  }
});

async function copyParentEmailToClipboard() {
  const email = current.value.email;
  if (!email || typeof navigator === 'undefined' || !navigator.clipboard) return;

  try {
    await navigator.clipboard.writeText(email);
    isParentEmailCopied.value = true;

    if (parentEmailCopyResetTimer) {
      clearTimeout(parentEmailCopyResetTimer);
    }

    parentEmailCopyResetTimer = setTimeout(() => {
      isParentEmailCopied.value = false;
      parentEmailCopyResetTimer = null;
    }, 2000);
  } catch {
    isParentEmailCopied.value = false;
  }
}

function startEditingParent() {
  parentEditForm.value = {
    firstName: current.value.firstName,
    lastName: current.value.lastName,
    birthDate: current.value.birthDate ? formatDateForInput(current.value.birthDate) : undefined,
    email: current.value.email,
    phone: current.value.phone,
    street: current.value.street,
    streetNo: current.value.streetNo,
    postalCode: current.value.postalCode,
    city: current.value.city,
  };
  isEditingParent.value = true;
}

function cancelEditingParent() {
  isEditingParent.value = false;
  parentEditForm.value = {};
  parentDetailError.value = null;
}

async function saveParentEdit() {
  isSavingParent.value = true;
  parentDetailError.value = null;
  try {
    current.value = await api.updateParent(current.value.id, parentEditForm.value);
    isEditingParent.value = false;
    // Lets the page reload the child so the parent list is up to date
    emit('saved');
  } catch (e) {
    parentDetailError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isSavingParent.value = false;
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center">
            <User class="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 class="text-xl font-semibold">
              {{ current.firstName }} {{ current.lastName }}
            </h2>
            <p class="text-sm text-gray-500">Elternteil</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button
            v-if="!isEditingParent"
            @click="startEditingParent"
            class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            title="Bearbeiten"
          >
            <Edit class="h-5 w-5" />
          </button>
            <button @click="$emit('close')" class="p-1 hover:bg-gray-100 rounded" aria-label="Schließen">
              <X class="h-5 w-5" />
            </button>
        </div>
      </div>

      <!-- View Mode -->
      <div v-if="!isEditingParent" class="space-y-4">
        <div v-if="current.birthDate">
          <p class="text-sm text-gray-500">Geburtsdatum</p>
          <p class="font-medium">{{ formatDate(current.birthDate) }}</p>
        </div>

        <div v-if="current.email">
          <p class="text-sm text-gray-500">E-Mail</p>
          <div class="mt-1 flex items-center gap-2">
            <a :href="`mailto:${current.email}`" class="font-medium text-primary hover:underline break-all">
              {{ current.email }}
            </a>
            <button
              type="button"
              @click="copyParentEmailToClipboard"
              class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
              :title="isParentEmailCopied ? 'E-Mail kopiert' : 'E-Mail kopieren'"
            >
              <Check v-if="isParentEmailCopied" class="h-3.5 w-3.5" />
              <Copy v-else class="h-3.5 w-3.5" />
              {{ isParentEmailCopied ? 'Kopiert' : 'Kopieren' }}
            </button>
          </div>
        </div>

        <div v-if="current.phone">
          <p class="text-sm text-gray-500">Telefon</p>
          <a :href="`tel:${current.phone}`" class="font-medium text-primary hover:underline">
            {{ current.phone }}
          </a>
        </div>

        <div v-if="current.street">
          <p class="text-sm text-gray-500">Adresse</p>
          <p class="font-medium">{{ current.street }} {{ current.streetNo }}</p>
          <p class="text-gray-600">{{ current.postalCode }} {{ current.city }}</p>
        </div>

        <div class="pt-4 border-t text-sm text-gray-500">
          <p>Erstellt: {{ formatDate(current.createdAt) }}</p>
          <p>Aktualisiert: {{ formatDate(current.updatedAt) }}</p>
        </div>
      </div>

      <!-- Edit Mode -->
      <form v-else @submit.prevent="saveParentEdit" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="parent-edit-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
            <input
              id="parent-edit-firstName"
              v-model="parentEditForm.firstName"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="parent-edit-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
            <input
              id="parent-edit-lastName"
              v-model="parentEditForm.lastName"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div>
          <label for="parent-edit-birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
          <input
            id="parent-edit-birthDate"
            v-model="parentEditForm.birthDate"
            type="date"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div>
          <label for="parent-edit-email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
          <input
            id="parent-edit-email"
            v-model="parentEditForm.email"
            type="email"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div>
          <label for="parent-edit-phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
          <input
            id="parent-edit-phone"
            v-model="parentEditForm.phone"
            type="tel"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div class="grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="parent-edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              id="parent-edit-street"
              v-model="parentEditForm.street"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="parent-edit-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
            <input
              id="parent-edit-streetNo"
              v-model="parentEditForm.streetNo"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div>
            <label for="parent-edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
            <input
              id="parent-edit-postalCode"
              v-model="parentEditForm.postalCode"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div class="col-span-2">
            <label for="parent-edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
            <input
              id="parent-edit-city"
              v-model="parentEditForm.city"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div v-if="parentDetailError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ parentDetailError }}</p>
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            @click="cancelEditingParent"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            :disabled="isSavingParent"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isSavingParent" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
            Speichern
          </button>
        </div>
      </form>
    </div>
  </div></template>
