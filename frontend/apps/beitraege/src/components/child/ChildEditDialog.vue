<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { CareHoursHistoryEntry, Child, ChildEinstufungSync, LegalHoursHistoryEntry, UpdateChildRequest } from '@/api/types';
import { Loader2, Check } from 'lucide-vue-next';
import { formatDateForInput } from '@/utils/format';
import AppSheet from '@/components/ui/AppSheet.vue';

const props = defineProps<{
  child: Child;
  careHoursHistory: CareHoursHistoryEntry[];
  legalHoursHistory: LegalHoursHistoryEntry[];
}>();

const emit = defineEmits<{ close: []; saved: [sync: ChildEinstufungSync | null] }>();

// Child data and care hours are saved in two requests; report their effects together.
function mergeSync(a: ChildEinstufungSync | null, b: ChildEinstufungSync | null | undefined): ChildEinstufungSync | null {
  if (!a || !b) return a ?? b ?? null;
  const einstufungen = [...a.einstufungen];
  for (const e of b.einstufungen) {
    const known = einstufungen.find((x) => x.id === e.id);
    if (known) known.letterSent ||= e.letterSent;
    else einstufungen.push(e);
  }
  const x = a.expectationChanges;
  const y = b.expectationChanges;
  return {
    einstufungen,
    expectationChanges: {
      created: x.created + y.created,
      updated: x.updated + y.updated,
      removed: x.removed + y.removed,
      skipped: x.skipped + y.skipped,
      deltaOpen: x.deltaOpen + y.deltaOpen,
      creditReviewRequired: [...x.creditReviewRequired, ...y.creditReviewRequired],
    },
  };
}

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

    const updated = await api.updateChild(props.child.id, childUpdate);
    let sync = mergeSync(null, updated.einstufungSync);
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
      sync = mergeSync(sync, await api.addCareHoursHistory(props.child.id, {
        careHours: nextCareHours,
        validFrom: careHoursValidFrom,
      }));
    }
    emit('saved', sync);
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
  }
}
</script>

<template>
  <AppSheet title="Kind bearbeiten" size="md" @close="$emit('close')">
      <form id="edit-child-form" @submit.prevent="handleEdit" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="edit-firstName" class="label">Vorname</label>
            <input
              id="edit-firstName"
              v-model="editForm.firstName"
              type="text"
              class="input"
            />
          </div>
          <div>
            <label for="edit-lastName" class="label">Nachname</label>
            <input
              id="edit-lastName"
              v-model="editForm.lastName"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="edit-birthDate" class="label">Geburtsdatum</label>
            <input
              id="edit-birthDate"
              v-model="editForm.birthDate"
              type="date"
              class="input"
            />
          </div>
          <div>
            <label for="edit-entryDate" class="label">Eintrittsdatum</label>
            <input
              id="edit-entryDate"
              v-model="editForm.entryDate"
              type="date"
              class="input"
            />
          </div>
        </div>

        <div>
          <label for="edit-exitDate" class="label">Austrittsdatum</label>
          <input
            id="edit-exitDate"
            v-model="editForm.exitDate"
            type="date"
            class="input"
          />
          <p class="text-xs text-muted-foreground mt-1">Optional: Datum, an dem das Kind die Kita verlässt</p>
        </div>

        <div class="grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="edit-street" class="label">Straße</label>
            <input
              id="edit-street"
              v-model="editForm.street"
              type="text"
              class="input"
            />
          </div>
          <div>
            <label for="edit-streetNo" class="label">Hausnr.</label>
            <input
              id="edit-streetNo"
              v-model="editForm.streetNo"
              type="text"
              class="input"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div>
            <label for="edit-postalCode" class="label">PLZ</label>
            <input
              id="edit-postalCode"
              v-model="editForm.postalCode"
              type="text"
              class="input"
            />
          </div>
          <div class="col-span-2">
            <label for="edit-city" class="label">Ort</label>
            <input
              id="edit-city"
              v-model="editForm.city"
              type="text"
              class="input"
            />
          </div>
        </div>

        <!-- Care Hours Section -->
        <div class="pt-4 border-t">
          <h3 class="text-sm font-medium text-foreground mb-3">Betreuungszeiten</h3>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-legalHours" class="label">Rechtsanspruch (Std./Woche)</label>
              <input
                id="edit-legalHours"
                v-model.number="editForm.legalHours"
                type="number"
                min="0"
                max="50"
                class="input"
              />
            </div>
            <div>
              <label for="edit-legalHoursValidFrom" class="label">Rechtsanspruch gültig ab</label>
              <input
                id="edit-legalHoursValidFrom"
                v-model="editForm.legalHoursValidFrom"
                type="date"
                class="input"
              />
            </div>
          </div>
          <div class="mt-4">
            <label for="edit-careHours" class="label">Betreuungszeit (Std./Woche)</label>
            <input
              id="edit-careHours"
              v-model.number="editForm.careHours"
              type="number"
              min="0"
              max="50"
              class="input"
            />
            <p class="text-xs text-muted-foreground mt-1">Änderungen werden als Historieneintrag gespeichert und gelten ab dem gewählten Datum.</p>
          </div>
          <div class="mt-4">
            <label for="edit-careHoursValidFrom" class="label">Betreuungszeit gültig ab</label>
            <input
              id="edit-careHoursValidFrom"
              v-model="editForm.careHoursValidFrom"
              type="date"
              class="input"
            />
          </div>
        </div>

        <div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="editForm.isActive"
              type="checkbox"
              class="w-4 h-4 text-primary rounded border-border focus:ring-primary"
            />
            <span class="text-sm text-foreground">Kind ist aktiv</span>
          </label>
        </div>

        <div v-if="editError" class="note note-danger">
          <p class="text-sm text-destructive">{{ editError }}</p>
        </div>

      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="$emit('close')">Abbrechen</button>
        <button type="submit" form="edit-child-form" class="btn btn-primary" :disabled="isEditing">
          <Loader2 v-if="isEditing" class="animate-spin" /><Check v-else />Speichern
        </button>
      </template>
  </AppSheet>
</template>
