<script setup lang="ts">
import { computed, toRefs } from 'vue';
import type { ReminderCase, ReminderCaseFee, ReminderCasePreview } from '@/api/types';
import { formatCurrency, formatDate } from '@/utils/format';
import { feeChipClass, feeTypeLabel, formatPeriod, statusBadgeClass, statusLabel } from '@/utils/reminders';

const props = defineProps<{
  selectedCase: ReminderCase;
  isNestedReminder: (fee: ReminderCaseFee) => boolean;
  hasNestedReminderBelow: (index: number) => boolean;
  selectedFeeIds: string[];
  reminderFeeIds: string[];
  includeQR: boolean;
  preview: ReminderCasePreview | null;
  isPreviewLoading: boolean;
  previewError: string | null;
  subjectEdit: string;
  bodyEdit: string;
  userEdited: boolean;
  resetNotice: boolean;
  isSending: boolean;
  sendError: string | null;
  conflictFeeCount: number;
  isMahnung: boolean;
  toggleFee: (feeId: string) => void;
  toggleReminderFee: (feeId: string) => void;
  onSubjectInput: () => void;
  onBodyInput: () => void;
  resetEdits: () => void;
  openSendConfirmation: () => Promise<void>;
}>();

const {
  selectedCase,
  isNestedReminder,
  hasNestedReminderBelow,
  selectedFeeIds,
  reminderFeeIds,
  includeQR,
  preview,
  isPreviewLoading,
  previewError,
  userEdited,
  resetNotice,
  isSending,
  sendError,
  conflictFeeCount,
  isMahnung,
  toggleFee,
  toggleReminderFee,
  onSubjectInput,
  onBodyInput,
  resetEdits,
  openSendConfirmation,
} = toRefs(props);

const emit = defineEmits<{
  'update:subjectEdit': [value: string];
  'update:bodyEdit': [value: string];
}>();
const subjectEdit = computed({
  get: () => props.subjectEdit,
  set: (value) => emit('update:subjectEdit', value),
});
const bodyEdit = computed({
  get: () => props.bodyEdit,
  set: (value) => emit('update:bodyEdit', value),
});
</script>

<template>
  <!-- Fee selection -->
  <div class="border rounded-lg overflow-hidden mb-4">
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-muted-foreground border-b bg-muted">
            <th class="w-8 py-2 pl-3"></th>
            <th class="py-2 pr-3 font-medium">Kind/Mitglied · Beitrag</th>
            <th class="py-2 pr-3 font-medium">Zeitraum</th>
            <th class="py-2 pr-3 font-medium">Fällig</th>
            <th class="py-2 pr-3 font-medium text-right">Soll</th>
            <th class="py-2 pr-3 font-medium text-right">Offen</th>
            <th class="py-2 pr-3 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(fee, index) in selectedCase.fees"
            :key="fee.feeId"
            :class="hasNestedReminderBelow(index) ? '' : 'border-b last:border-0'"
          >
            <td class="py-2 pl-3">
              <input
                type="checkbox"
                :checked="selectedFeeIds.includes(fee.feeId)"
                @change="toggleFee(fee.feeId)"
              />
            </td>
            <td class="py-2 pr-3">
              <div class="flex items-center gap-2">
                <router-link
                  v-if="fee.clubMember && !isNestedReminder(fee)"
                  :to="`/mitglieder/${fee.clubMember.id}`"
                  class="text-primary hover:underline whitespace-nowrap"
                  :title="`Vereinsmitglied ${fee.clubMember.memberNumber} öffnen`"
                >
                  {{ fee.clubMember.name }}
                </router-link>
                <router-link
                  v-else-if="!isNestedReminder(fee)"
                  :to="`/kinder/${fee.childId}`"
                  class="text-primary hover:underline whitespace-nowrap"
                  title="Kind öffnen"
                >
                  {{ fee.childName }}
                </router-link>
                <span class="px-2 py-0.5 text-xs rounded-full font-medium" :class="feeChipClass(fee.feeType)">
                  {{ feeTypeLabel(fee.feeType) }}
                </span>
              </div>
            </td>
            <td class="py-2 pr-3 whitespace-nowrap">{{ formatPeriod(fee) }}</td>
            <td class="py-2 pr-3 whitespace-nowrap">{{ formatDate(fee.dueDate) }}</td>
            <td class="py-2 pr-3 text-right whitespace-nowrap">{{ formatCurrency(fee.amount) }}</td>
            <td class="py-2 pr-3 text-right font-medium whitespace-nowrap">{{ formatCurrency(fee.remaining) }}</td>
            <td class="py-2 pr-3">
              <span class="px-2 py-0.5 text-xs rounded-full font-medium whitespace-nowrap" :class="statusBadgeClass(fee.status)">
                {{ statusLabel(fee.status) }}
              </span>
              <label
                v-if="fee.reminderFeeDue"
                class="mt-1 flex items-center gap-1.5 text-xs whitespace-nowrap"
                :class="reminderFeeIds.includes(fee.feeId) ? 'text-amber-700 dark:text-amber-300 font-medium' : 'text-foreground'"
                :title="selectedFeeIds.includes(fee.feeId) ? 'Laut Regeln fällig – wird nur mit Häkchen erhoben' : 'Erst den Beitrag auswählen'"
              >
                <input
                  type="checkbox"
                  :checked="reminderFeeIds.includes(fee.feeId)"
                  :disabled="!selectedFeeIds.includes(fee.feeId)"
                  @change="toggleReminderFee(fee.feeId)"
                />
                Mahngebühr erheben
              </label>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>

  <!-- Warnings -->
  <div
    v-if="preview && preview.warnings && preview.warnings.length > 0"
    class="p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-300 rounded-lg text-sm text-amber-900 dark:text-amber-300 mb-4"
  >
    <p class="font-semibold mb-1">Hinweise</p>
    <ul class="space-y-0.5">
      <li v-for="(warning, index) in preview.warnings" :key="index">· {{ warning }}</li>
    </ul>
  </div>

  <!-- Send error -->
  <div v-if="sendError" class="p-3 bg-red-50 dark:bg-red-950/40 border border-red-300 rounded-lg text-sm text-red-800 dark:text-red-300 mb-4">
    {{ sendError }}
    <span v-if="conflictFeeCount > 0" class="block text-xs mt-1">
      Betroffene Beiträge: {{ conflictFeeCount }} — bitte Auswahl prüfen.
    </span>
  </div>

  <!-- Preview -->
  <div v-if="selectedFeeIds.length === 0" class="text-sm text-muted-foreground mb-4">
    Bitte mindestens einen Beitrag auswählen.
  </div>
  <div v-else-if="isPreviewLoading && !preview" class="text-sm text-muted-foreground mb-4">Vorschau wird erstellt...</div>
  <div v-else-if="previewError" class="text-sm text-red-600 dark:text-red-300 mb-4">{{ previewError }}</div>
  <div v-else-if="preview" class="border rounded-lg p-4 mb-4 bg-muted">
    <div class="flex flex-wrap items-center justify-between gap-2 mb-3">
      <p class="text-sm font-medium text-foreground">
        Vorschau · Frist: <span class="font-semibold">{{ formatDate(preview.deadline) }}</span>
        · Gesamtbetrag: <span class="font-semibold">{{ formatCurrency(preview.totalAmount) }}</span>
      </p>
      <p v-if="resetNotice" class="text-xs text-amber-700 dark:text-amber-300">Manuelle Textänderungen wurden zurückgesetzt.</p>
    </div>

    <!-- Planned reminder fees -->
    <div v-if="preview.plannedReminderFees.length > 0" class="mb-3 p-3 bg-card border rounded-lg text-sm">
      <p class="font-medium text-foreground mb-1">Neu entstehende Mahngebühren</p>
      <ul class="space-y-0.5 text-foreground">
        <li v-for="planned in preview.plannedReminderFees" :key="planned.baseFeeId" class="flex justify-between gap-3">
          <span>Mahngebühr für {{ planned.baseLabel }}</span>
          <span class="font-medium">{{ formatCurrency(planned.amount) }}</span>
        </li>
      </ul>
    </div>

    <div class="space-y-2">
      <div>
        <label class="block text-xs text-muted-foreground mb-1">Betreff</label>
        <input
          type="text"
          v-model="subjectEdit"
          @input="onSubjectInput"
          class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none bg-card"
        />
      </div>
      <div>
        <label class="block text-xs text-muted-foreground mb-1">Text</label>
        <textarea
          v-model="bodyEdit"
          @input="onBodyInput"
          rows="14"
          class="w-full px-3 py-2 border border-border rounded-lg font-mono text-xs focus:ring-2 focus:ring-primary focus:border-transparent outline-none whitespace-pre-wrap bg-card"
        ></textarea>
        <p v-if="userEdited" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
          Text angepasst — Änderungen an der Auswahl oder den Mahngebühren setzen ihn zurück.
        </p>
      </div>
      <div v-if="includeQR && preview.qrImageDataUrl" class="flex flex-col sm:flex-row gap-3">
        <div class="shrink-0">
          <p class="text-xs text-muted-foreground mb-1">SEPA-QR-Code</p>
          <img :src="preview.qrImageDataUrl" alt="SEPA QR-Code" class="w-full max-w-[220px] border rounded bg-card p-2" />
        </div>
        <div v-if="preview.qrPayload" class="flex-1 min-w-0 flex flex-col">
          <p class="text-xs text-muted-foreground mb-1">Im QR-Code enthalten</p>
          <pre class="flex-1 whitespace-pre-wrap break-all font-mono text-xs text-muted-foreground bg-card border rounded p-3">{{ preview.qrPayload }}</pre>
        </div>
      </div>
      <p v-else-if="!includeQR" class="text-xs text-muted-foreground">QR-Code ist deaktiviert und wird nicht angehängt.</p>
    </div>
  </div>

  <!-- Actions -->
  <div class="flex flex-wrap justify-end gap-3">
    <button
      class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-accent"
      @click="resetEdits"
      v-if="preview && userEdited"
    >
      Text zurücksetzen
    </button>
    <button
      class="px-4 py-2 bg-primary text-primary-foreground text-sm font-medium rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
      :disabled="!preview || isPreviewLoading || isSending || selectedCase.recipients.length === 0"
      @click="openSendConfirmation"
    >
      {{ isMahnung ? 'Mahnung senden' : 'Erinnerung senden' }}
    </button>
  </div>
</template>
