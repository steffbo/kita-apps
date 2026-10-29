<script setup lang="ts">
import { ref, onUnmounted } from 'vue';
import { api } from '@/api';
import type { Parent, UpdateParentRequest } from '@/api/types';
import { Edit, Loader2, User, Check, Copy } from 'lucide-vue-next';
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
import AppSheet from '@/components/ui/AppSheet.vue';
</script>

<template>
  <AppSheet :title="`${current.firstName} ${current.lastName}`" hide-title @close="$emit('close')">

      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center">
            <User class="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 class="text-base font-semibold">
              {{ current.firstName }} {{ current.lastName }}
            </h2>
            <p class="text-sm text-muted-foreground">Elternteil</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button
            v-if="!isEditingParent"
            @click="startEditingParent"
            class="p-2 text-muted-foreground hover:text-foreground hover:bg-accent rounded-lg transition-colors"
            title="Bearbeiten"
          >
            <Edit class="h-5 w-5" />
          </button>
        </div>
      </div>

      <!-- View Mode -->
      <div v-if="!isEditingParent" class="space-y-4">
        <div v-if="current.birthDate">
          <p class="text-sm text-muted-foreground">Geburtsdatum</p>
          <p class="font-medium">{{ formatDate(current.birthDate) }}</p>
        </div>

        <div v-if="current.email">
          <p class="text-sm text-muted-foreground">E-Mail</p>
          <div class="mt-1 flex items-center gap-2">
            <a :href="`mailto:${current.email}`" class="font-medium text-primary hover:underline break-all">
              {{ current.email }}
            </a>
            <button
              type="button"
              @click="copyParentEmailToClipboard"
              class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-foreground bg-muted hover:bg-accent rounded-md transition-colors"
              :title="isParentEmailCopied ? 'E-Mail kopiert' : 'E-Mail kopieren'"
            >
              <Check v-if="isParentEmailCopied" class="h-3.5 w-3.5" />
              <Copy v-else class="h-3.5 w-3.5" />
              {{ isParentEmailCopied ? 'Kopiert' : 'Kopieren' }}
            </button>
          </div>
        </div>

        <div v-if="current.phone">
          <p class="text-sm text-muted-foreground">Telefon</p>
          <a :href="`tel:${current.phone}`" class="font-medium text-primary hover:underline">
            {{ current.phone }}
          </a>
        </div>

        <div v-if="current.street">
          <p class="text-sm text-muted-foreground">Adresse</p>
          <p class="font-medium">{{ current.street }} {{ current.streetNo }}</p>
          <p class="text-muted-foreground">{{ current.postalCode }} {{ current.city }}</p>
        </div>

        <div class="pt-4 border-t text-sm text-muted-foreground">
          <p>Erstellt: {{ formatDate(current.createdAt) }}</p>
          <p>Aktualisiert: {{ formatDate(current.updatedAt) }}</p>
        </div>
      </div>

      <!-- Edit Mode -->
      <form id="parent-edit-form" v-else @submit.prevent="saveParentEdit" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="parent-edit-firstName" class="label">Vorname</label>
            <input
              id="parent-edit-firstName"
              v-model="parentEditForm.firstName"
              type="text"
              class="input"
            />
          </div>
          <div>
            <label for="parent-edit-lastName" class="label">Nachname</label>
            <input
              id="parent-edit-lastName"
              v-model="parentEditForm.lastName"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div>
          <label for="parent-edit-birthDate" class="label">Geburtsdatum</label>
          <input
            id="parent-edit-birthDate"
            v-model="parentEditForm.birthDate"
            type="date"
            class="input"
          />
        </div>

        <div>
          <label for="parent-edit-email" class="label">E-Mail</label>
          <input
            id="parent-edit-email"
            v-model="parentEditForm.email"
            type="email"
            class="input"
          />
        </div>

        <div>
          <label for="parent-edit-phone" class="label">Telefon</label>
          <input
            id="parent-edit-phone"
            v-model="parentEditForm.phone"
            type="tel"
            class="input"
          />
        </div>

        <div class="grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="parent-edit-street" class="label">Straße</label>
            <input
              id="parent-edit-street"
              v-model="parentEditForm.street"
              type="text"
              class="input"
            />
          </div>
          <div>
            <label for="parent-edit-streetNo" class="label">Hausnr.</label>
            <input
              id="parent-edit-streetNo"
              v-model="parentEditForm.streetNo"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div>
            <label for="parent-edit-postalCode" class="label">PLZ</label>
            <input
              id="parent-edit-postalCode"
              v-model="parentEditForm.postalCode"
              type="text"
              class="input"
            />
          </div>
          <div class="col-span-2">
            <label for="parent-edit-city" class="label">Ort</label>
            <input
              id="parent-edit-city"
              v-model="parentEditForm.city"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div v-if="parentDetailError" class="note note-danger">
          <p class="text-sm text-destructive">{{ parentDetailError }}</p>
        </div>

      </form>
      <template v-if="isEditingParent" #footer>
        <button type="button" class="btn btn-secondary" @click="cancelEditingParent">Abbrechen</button>
        <button type="submit" form="parent-edit-form" class="btn btn-primary" :disabled="isSavingParent">
          <Loader2 v-if="isSavingParent" class="animate-spin" /><Check v-else />Speichern
        </button>
      </template>
  </AppSheet>
</template>
