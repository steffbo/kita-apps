<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { api } from '@/api';
import type { InstallmentDetail, InstallmentReminderPreview } from '@/api/types';
import { AlertTriangle, Loader2, Send } from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import DetailRow from '@/components/ui/DetailRow.vue';
import { formatCurrency, formatDate } from '@/utils/format';
import { overdueRates } from '@/utils/installments';

// Zahlungserinnerung or Mahnung for overdue rates. The server builds the text; the user may edit it before sending.
const props = defineProps<{ detail: InstallmentDetail }>();
const emit = defineEmits<{ close: []; sent: []; edit: [] }>();

const overdue = computed(() => overdueRates(props.detail));
const selected = ref<number[]>(overdue.value.map(rate => rate.number));
const includeQR = ref(true);
const chargeFee = ref(false);
const preview = ref<InstallmentReminderPreview | null>(null);
const subject = ref('');
const body = ref('');
const isLoading = ref(false);
const previewError = ref<string | null>(null);
const isSending = ref(false);
const sendError = ref<string | null>(null);
let requestSeq = 0;

const isMahnung = computed(() => preview.value?.stage === 'final');
// A Mahngebühr is only possible in the final stage, so the server never sees it for a first reminder.
const feeChargeable = computed(() => isMahnung.value && chargeFee.value);
const hasEmail = computed(() => props.detail.plan.emails.length > 0);
const canSend = computed(() => !!preview.value && preview.value.recipients.length > 0
  && selected.value.length > 0 && !isLoading.value && !isSending.value);

function toggleRate(number: number): void {
  selected.value = selected.value.includes(number)
    ? selected.value.filter(item => item !== number)
    : [...selected.value, number].sort((a, b) => a - b);
}

// Only the latest request may update the preview, otherwise a slow older response overwrites newer text.
async function refreshPreview(): Promise<void> {
  const seq = ++requestSeq;
  if (selected.value.length === 0) {
    preview.value = null;
    isLoading.value = false;
    return;
  }
  isLoading.value = true;
  previewError.value = null;
  try {
    const result = await api.previewInstallmentReminder(props.detail.plan.id, {
      installmentNumbers: [...selected.value],
      chargeReminderFee: feeChargeable.value,
      includeQR: includeQR.value,
    });
    if (seq !== requestSeq) return;
    preview.value = result;
    subject.value = result.subject;
    body.value = result.body;
  } catch (e) {
    if (seq !== requestSeq) return;
    preview.value = null;
    previewError.value = e instanceof Error ? e.message : 'Vorschau konnte nicht geladen werden';
  } finally {
    if (seq === requestSeq) isLoading.value = false;
  }
}

async function send(): Promise<void> {
  if (!canSend.value) return;
  isSending.value = true;
  sendError.value = null;
  try {
    await api.sendInstallmentReminder(props.detail.plan.id, {
      installmentNumbers: [...selected.value],
      chargeReminderFee: feeChargeable.value,
      includeQR: includeQR.value,
      subject: subject.value,
      body: body.value,
    });
    emit('sent');
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : 'Versand fehlgeschlagen';
  } finally {
    isSending.value = false;
  }
}

watch([selected, includeQR, chargeFee], () => void refreshPreview(), { deep: true });
onMounted(() => void refreshPreview());
</script>

<template>
  <AppSheet :title="`${isMahnung ? 'Mahnung' : 'Zahlungserinnerung'} · ${detail.plan.debtorName}`" size="lg"
    @close="emit('close')">
    <div class="space-y-4">
      <div v-if="!hasEmail" class="note note-danger">
        <AlertTriangle />
        <span class="flex-1">Ohne E-Mail-Adresse kein Versand. Adresse ergänzen, um zu erinnern.</span>
        <button type="button" class="btn btn-secondary btn-sm" @click="emit('edit')">Ergänzen</button>
      </div>

      <div>
        <span class="label">Überfällige Raten</span>
        <ul class="card divide-y">
          <li v-for="rate in overdue" :key="rate.number">
            <label class="flex min-h-12 cursor-pointer items-center gap-3 px-4 py-2.5 hover:bg-accent">
              <input type="checkbox" class="h-4 w-4 rounded" :checked="selected.includes(rate.number)"
                @change="toggleRate(rate.number)" />
              <span class="min-w-0 flex-1">
                <span class="block font-medium">Rate {{ rate.number }} · {{ formatCurrency(rate.amount - rate.paid) }}</span>
                <span class="block text-xs text-muted-foreground">fällig {{ formatDate(rate.dueDate) }}</span>
              </span>
            </label>
          </li>
          <li v-if="isMahnung">
            <label class="flex min-h-12 cursor-pointer items-center gap-3 px-4 py-2.5 hover:bg-accent">
              <input v-model="chargeFee" type="checkbox" class="h-4 w-4 rounded" />
              <span class="font-medium">Mahngebühr erheben</span>
            </label>
          </li>
        </ul>
      </div>

      <label class="flex items-center gap-2 text-sm">
        <input v-model="includeQR" type="checkbox" class="h-4 w-4 rounded" />QR-Code (SEPA) anhängen
      </label>

      <div v-if="previewError" class="note note-danger" role="alert"><AlertTriangle />{{ previewError }}</div>
      <div v-else-if="isLoading && !preview" class="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 class="h-4 w-4 animate-spin" />Vorschau wird erstellt
      </div>

      <template v-if="preview">
        <dl class="card divide-y text-sm">
          <DetailRow label="Empfänger" value-class="break-all text-right">{{ preview.recipients.join(', ') || '—' }}</DetailRow>
          <DetailRow label="Frist">{{ formatDate(preview.deadline) }}</DetailRow>
          <DetailRow label="Gesamtbetrag" value-class="font-medium tabular-nums">{{ formatCurrency(preview.totalAmount) }}</DetailRow>
        </dl>

        <div v-if="preview.warnings.length" class="note note-warning">
          <AlertTriangle />
          <ul class="space-y-0.5">
            <li v-for="(warning, index) in preview.warnings" :key="index">{{ warning }}</li>
          </ul>
        </div>

        <div>
          <label for="reminder-subject" class="label">Betreff</label>
          <input id="reminder-subject" v-model="subject" type="text" class="input" />
        </div>
        <div>
          <label for="reminder-body" class="label">Text</label>
          <textarea id="reminder-body" v-model="body" rows="12" class="input font-mono text-xs" />
        </div>
        <img v-if="includeQR && preview.qrImageDataUrl" :src="preview.qrImageDataUrl" alt="SEPA QR-Code"
          class="w-44 rounded-lg border bg-card p-2" />
      </template>

      <div v-if="sendError" class="note note-danger" role="alert"><AlertTriangle />{{ sendError }}</div>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="button" class="btn btn-primary" :disabled="!canSend" @click="send">
        <Loader2 v-if="isSending" class="animate-spin" /><Send v-else />Senden
      </button>
    </template>
  </AppSheet>
</template>
