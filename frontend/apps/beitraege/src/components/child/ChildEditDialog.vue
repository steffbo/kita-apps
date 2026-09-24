<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { CareHoursHistoryEntry, Child, LegalHoursHistoryEntry, UpdateChildRequest } from '@/api/types';
import { Loader2, X, Check } from 'lucide-vue-next';
import { formatDateForInput } from '@/utils/format';

const props = defineProps<{
  child: Child;
  careHoursHistory: CareHoursHistoryEntry[];
  legalHoursHistory: LegalHoursHistoryEntry[];
}>();

const emit = defineEmits<{ close: []; saved: [] }>();

type UpdateChildForm = UpdateChildRequest & {
  legalHoursValidFrom?: string;
  careHoursValidFrom?: string;
};

function getLatestEffectiveFrom(
  history: Array<{ effectiveFrom: string }>,
  fallbackDate: string,
): string {
  const latestEntry = history[0];
  if (latestEntry?.effectiveFrom) {
    return formatDateForInput(latestEntry.effectiveFrom);
  }
  return formatDateForInput(fallbackDate);
}

function normalizeCareHoursValue(value: unknown): number | null {
  if (value === '' || value === undefined || value === null) return null;
  if (typeof value === 'number') {
    return Number.isNaN(value) ? null : value;
  }
  const parsed = Number(value);
  return Number.isNaN(parsed) ? null : parsed;
}

const editForm = ref<UpdateChildForm>({
  firstName: props.child.firstName,
  lastName: props.child.lastName,
  birthDate: formatDateForInput(props.child.birthDate),
  entryDate: formatDateForInput(props.child.entryDate),
  exitDate: props.child.exitDate ? formatDateForInput(props.child.exitDate) : undefined,
  street: props.child.street,
  streetNo: props.child.streetNo,
  postalCode: props.child.postalCode,
  city: props.child.city,
  legalHours: props.child.legalHours,
  legalHoursValidFrom: getLatestEffectiveFrom(props.legalHoursHistory, props.child.entryDate),
  careHours: props.child.careHours,
  careHoursValidFrom: getLatestEffectiveFrom(props.careHoursHistory, props.child.entryDate),
  isActive: props.child.isActive,
});
const isEditing = ref(false);
const editError = ref<string | null>(null);

async function handleEdit() {
  isEditing.value = true;
  editError.value = null;
  try {
    const originalCareHours = normalizeCareHoursValue(props.child.careHours);
    const nextCareHours = normalizeCareHoursValue(editForm.value.careHours);
    const careHoursChanged = originalCareHours !== nextCareHours;
    const originalLegalHours = normalizeCareHoursValue(props.child.legalHours);
    const nextLegalHours = normalizeCareHoursValue(editForm.value.legalHours);
    const legalHoursChanged = originalLegalHours !== nextLegalHours;
    const {
      legalHoursValidFrom,
      careHoursValidFrom,
      legalHours: _legalHours,
      legalHoursUntil: _legalHoursUntil,
      careHours: _careHours,
      ...childUpdate
    } = editForm.value;

    await api.updateChild(props.child.id, childUpdate);
    if (legalHoursChanged) {
      if (!legalHoursValidFrom) {
        throw new Error('Bitte ein Gültig-ab-Datum für den Rechtsanspruch angeben.');
      }
      await api.addLegalHoursHistory(props.child.id, {
        legalHours: nextLegalHours,
        validFrom: legalHoursValidFrom,
      });
    }
    if (careHoursChanged) {
      if (!careHoursValidFrom) {
        throw new Error('Bitte ein Gültig-ab-Datum für die Betreuungszeit angeben.');
      }
      await api.addCareHoursHistory(props.child.id, {
        careHours: nextCareHours,
        validFrom: careHoursValidFrom,
      });
    }
    emit('saved');
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
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
        <h2 class="text-xl font-semibold">Kind bearbeiten</h2>
        <button @click="$emit('close')" class="p-1 hover:bg-gray-100 rounded">
          <X class="h-5 w-5" />
        </button>
      </div>

      <form @submit.prevent="handleEdit" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="edit-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
            <input
              id="edit-firstName"
              v-model="editForm.firstName"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="edit-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
            <input
              id="edit-lastName"
              v-model="editForm.lastName"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="edit-birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
            <input
              id="edit-birthDate"
              v-model="editForm.birthDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="edit-entryDate" class="block text-sm font-medium text-gray-700 mb-1">Eintrittsdatum</label>
            <input
              id="edit-entryDate"
              v-model="editForm.entryDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div>
          <label for="edit-exitDate" class="block text-sm font-medium text-gray-700 mb-1">Austrittsdatum</label>
          <input
            id="edit-exitDate"
            v-model="editForm.exitDate"
            type="date"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
          <p class="text-xs text-gray-500 mt-1">Optional: Datum, an dem das Kind die Kita verlässt</p>
        </div>

        <div class="grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              id="edit-street"
              v-model="editForm.street"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label for="edit-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
            <input
              id="edit-streetNo"
              v-model="editForm.streetNo"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div>
            <label for="edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
            <input
              id="edit-postalCode"
              v-model="editForm.postalCode"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div class="col-span-2">
            <label for="edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
            <input
              id="edit-city"
              v-model="editForm.city"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <!-- Care Hours Section -->
        <div class="pt-4 border-t">
          <h3 class="text-sm font-medium text-gray-700 mb-3">Betreuungszeiten</h3>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-legalHours" class="block text-sm font-medium text-gray-700 mb-1">Rechtsanspruch (Std./Woche)</label>
              <input
                id="edit-legalHours"
                v-model.number="editForm.legalHours"
                type="number"
                min="0"
                max="50"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-legalHoursValidFrom" class="block text-sm font-medium text-gray-700 mb-1">Rechtsanspruch gültig ab</label>
              <input
                id="edit-legalHoursValidFrom"
                v-model="editForm.legalHoursValidFrom"
                type="date"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>
          <div class="mt-4">
            <label for="edit-careHours" class="block text-sm font-medium text-gray-700 mb-1">Betreuungszeit (Std./Woche)</label>
            <input
              id="edit-careHours"
              v-model.number="editForm.careHours"
              type="number"
              min="0"
              max="50"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
            <p class="text-xs text-gray-500 mt-1">Änderungen werden als Historieneintrag gespeichert und gelten ab dem gewählten Datum.</p>
          </div>
          <div class="mt-4">
            <label for="edit-careHoursValidFrom" class="block text-sm font-medium text-gray-700 mb-1">Betreuungszeit gültig ab</label>
            <input
              id="edit-careHoursValidFrom"
              v-model="editForm.careHoursValidFrom"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="editForm.isActive"
              type="checkbox"
              class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
            />
            <span class="text-sm text-gray-700">Kind ist aktiv</span>
          </label>
        </div>

        <div v-if="editError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ editError }}</p>
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
            :disabled="isEditing"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isEditing" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
            Speichern
          </button>
        </div>
      </form>
    </div>
  </div></template>
