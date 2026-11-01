<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { api } from '@/api';
import type {
  BankTransaction, Household, InstallmentDetail, InstallmentPlan, InstallmentRequest,
} from '@/api/types';
import { AlertTriangle, CalendarDays, Link2, Loader2, Plus, X } from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import HouseholdPicker from '@/components/HouseholdPicker.vue';
import { formatCurrency, formatDate, formatDateForInput, todayISO } from '@/utils/format';
import {
  INTERVAL_OPTIONS, debtorNameFromPayer, firstOfNextMonth, referenceFromPurpose, schedulePreviewText,
} from '@/utils/installments';

// Create (optionally prefilled from a bank transaction, which becomes the first payment) or edit a plan.
const props = defineProps<{ plan?: InstallmentDetail; prefill?: BankTransaction }>();
const emit = defineEmits<{ close: []; saved: [planId: string] }>();

const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;

interface Draft {
  kind: 'household' | 'external';
  householdId: string | null;
  debtorName: string;
  debtorAddress: string;
  emails: string[];
  totalAmount: number | '';
  installmentAmount: number | '';
  intervalMonths: number;
  firstDueDate: string;
  reference: string;
  payerIban: string;
  note: string;
}

function draftFromPlan(plan: InstallmentPlan): Draft {
  return {
    kind: plan.householdId ? 'household' : 'external',
    householdId: plan.householdId ?? null,
    debtorName: plan.debtorName,
    debtorAddress: plan.debtorAddress,
    emails: [...plan.emails],
    totalAmount: plan.totalAmount,
    installmentAmount: plan.installmentAmount,
    intervalMonths: plan.intervalMonths,
    firstDueDate: formatDateForInput(plan.firstDueDate),
    reference: plan.reference,
    payerIban: plan.payerIban ?? '',
    note: plan.note,
  };
}

function draftFromTransaction(tx: BankTransaction): Draft {
  return {
    kind: 'external',
    householdId: null,
    debtorName: debtorNameFromPayer(tx.payerName ?? ''),
    debtorAddress: '',
    emails: [],
    totalAmount: '',
    installmentAmount: tx.amount,
    intervalMonths: 1,
    firstDueDate: formatDateForInput(tx.bookingDate),
    reference: referenceFromPurpose(tx.description ?? ''),
    payerIban: tx.payerIban ?? '',
    note: '',
  };
}

function blankDraft(): Draft {
  return {
    kind: 'household',
    householdId: null,
    debtorName: '',
    debtorAddress: '',
    emails: [],
    totalAmount: '',
    installmentAmount: '',
    intervalMonths: 1,
    firstDueDate: firstOfNextMonth(todayISO()),
    reference: '',
    payerIban: '',
    note: '',
  };
}

const form = reactive<Draft>(
  props.plan ? draftFromPlan(props.plan.plan)
    : props.prefill ? draftFromTransaction(props.prefill)
      : blankDraft(),
);
// Set after the first save, so a retry after a failed contract upload updates instead of creating again.
const planId = ref<string | null>(props.plan?.plan.id ?? null);
const contract = ref<File | null>(null);
const error = ref<string | null>(null);
const isSaving = ref(false);

const assignsTransaction = computed(() => !!props.prefill && !planId.value);
const title = computed(() => (props.plan ? 'Ratenzahlung bearbeiten' : 'Neue Ratenzahlung'));
const previewText = computed(() => schedulePreviewText(
  Number(form.totalAmount), Number(form.installmentAmount), form.intervalMonths, form.firstDueDate,
));

// ── Schuldner ───────────────────────────────────────────────────────────────
function setKind(kind: Draft['kind']): void {
  form.kind = kind;
  if (kind === 'external') form.householdId = null;
}

// Picking a household takes its name and the e-mail addresses of its parents.
async function selectHousehold(household: Household): Promise<void> {
  form.householdId = household.id;
  form.debtorName = household.name;
  error.value = null;
  try {
    const detail = await api.getHousehold(household.id);
    if (form.householdId !== household.id) return;
    const addresses = (detail.parents ?? []).flatMap(parent => (parent.email ? [parent.email] : []));
    form.emails = [...new Set(addresses)];
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Eltern konnten nicht geladen werden';
  }
}

// ── E-Mail-Adressen ─────────────────────────────────────────────────────────
const newEmail = ref('');

function addEmail(): void {
  const value = newEmail.value.trim();
  if (!EMAIL_PATTERN.test(value)) {
    error.value = 'Keine gültige E-Mail-Adresse';
    return;
  }
  if (!form.emails.includes(value)) form.emails.push(value);
  newEmail.value = '';
  error.value = null;
}

function removeEmail(index: number): void {
  form.emails.splice(index, 1);
}

// ── Speichern ───────────────────────────────────────────────────────────────
function pickContract(event: Event): void {
  contract.value = (event.target as HTMLInputElement).files?.[0] ?? null;
}

function toRequest(): InstallmentRequest {
  const isHousehold = form.kind === 'household';
  return {
    householdId: isHousehold ? form.householdId ?? undefined : undefined,
    debtorName: form.debtorName.trim(),
    debtorAddress: isHousehold ? '' : form.debtorAddress.trim(),
    emails: [...form.emails],
    totalAmount: Number(form.totalAmount),
    installmentAmount: Number(form.installmentAmount),
    intervalMonths: form.intervalMonths,
    firstDueDate: form.firstDueDate,
    reference: form.reference.trim(),
    payerIban: form.payerIban.trim(),
    note: form.note,
    // Only the create request may assign the bank transaction; updates reject it.
    transactionId: assignsTransaction.value ? props.prefill?.id : undefined,
  };
}

async function submit(): Promise<void> {
  error.value = null;
  if (form.kind === 'household' && !form.householdId) {
    error.value = 'Bitte Haushalt auswählen';
    return;
  }
  isSaving.value = true;
  try {
    const request = toRequest();
    const saved = planId.value
      ? await api.updateInstallmentPlan(planId.value, request)
      : await api.createInstallmentPlan(request);
    planId.value = saved.plan.id;
    if (contract.value) await api.uploadInstallmentContract(saved.plan.id, contract.value);
    emit('saved', saved.plan.id);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <AppSheet :title="title" @close="emit('close')">
    <form id="installment-plan-form" class="space-y-4" @submit.prevent="submit">
      <div v-if="assignsTransaction && prefill" class="note note-info">
        <Link2 />
        <div>
          Die Überweisung vom {{ formatDate(prefill.bookingDate) }} über {{ formatCurrency(prefill.amount) }}
          wird als erste Zahlung zugeordnet.
        </div>
      </div>

      <div>
        <span class="label">Schuldner</span>
        <div class="flex gap-2" role="group" aria-label="Schuldner">
          <button type="button" class="chip" :aria-pressed="form.kind === 'household'"
            @click="setKind('household')">Haushalt</button>
          <button type="button" class="chip" :aria-pressed="form.kind === 'external'"
            @click="setKind('external')">Andere Person</button>
        </div>
      </div>

      <HouseholdPicker v-if="form.kind === 'household'" v-model="form.householdId" @select="selectHousehold"
        @error="error = $event" />

      <div v-else class="space-y-4">
        <div>
          <label for="installment-name" class="label">Name *</label>
          <input id="installment-name" v-model="form.debtorName" required type="text" class="input"
            placeholder="z. B. Max Mustermann" />
        </div>
        <div>
          <label for="installment-address" class="label">Anschrift <span class="text-muted-foreground">(optional, für Briefe)</span></label>
          <textarea id="installment-address" v-model="form.debtorAddress" rows="2" class="input"
            placeholder="Straße, PLZ Ort" />
        </div>
      </div>

      <div>
        <span class="label">E-Mail-Adressen <span class="text-muted-foreground">(optional)</span></span>
        <div v-if="form.emails.length" class="mb-2 flex flex-wrap gap-1.5">
          <span v-for="(email, index) in form.emails" :key="email" class="badge h-7 gap-1 pl-2.5 pr-1">
            {{ email }}
            <button type="button" class="grid h-5 w-5 place-items-center rounded-full hover:bg-accent"
              :aria-label="`${email} entfernen`" @click="removeEmail(index)"><X class="h-3 w-3" /></button>
          </span>
        </div>
        <div class="flex gap-2">
          <input v-model="newEmail" type="email" inputmode="email" placeholder="name@beispiel.de" class="input"
            aria-label="E-Mail-Adresse" @keydown.enter.prevent="addEmail" />
          <button type="button" class="btn btn-secondary shrink-0" @click="addEmail">
            <Plus />Hinzufügen
          </button>
        </div>
        <p class="mt-1.5 text-xs text-muted-foreground">
          {{ form.kind === 'household' ? 'Vorbelegt mit den Adressen der Eltern. ' : '' }}Ohne Adresse erscheint der
          Fall in den Erinnerungen, kann aber nicht per E-Mail gemahnt werden.
        </p>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="installment-total" class="label">Gesamtbetrag *</label>
          <input id="installment-total" v-model.number="form.totalAmount" required type="number" min="0.01"
            step="0.01" inputmode="decimal" class="input" placeholder="0,00" />
        </div>
        <div>
          <label for="installment-rate" class="label">Ratenhöhe *</label>
          <input id="installment-rate" v-model.number="form.installmentAmount" required type="number" min="0.01"
            step="0.01" inputmode="decimal" class="input" placeholder="0,00" />
        </div>
      </div>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="installment-interval" class="label">Intervall</label>
          <select id="installment-interval" v-model.number="form.intervalMonths" class="input">
            <option v-for="option in INTERVAL_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>
        <div>
          <label for="installment-first" class="label">Erste Rate fällig *</label>
          <input id="installment-first" v-model="form.firstDueDate" required type="date" class="input" />
        </div>
      </div>
      <div class="note">
        <CalendarDays />
        <span>{{ previewText ?? 'Gesamtbetrag, Ratenhöhe und erste Fälligkeit eingeben' }}</span>
      </div>

      <div>
        <label for="installment-reference" class="label">Bezug / Aktenzeichen <span class="text-muted-foreground">(optional)</span></label>
        <input id="installment-reference" v-model="form.reference" type="text" class="input"
          placeholder="z. B. Rückstand Beiträge 2025" />
        <p class="mt-1.5 text-xs text-muted-foreground">Steht in Erinnerungen und hilft beim automatischen Abgleich.</p>
      </div>
      <div>
        <label for="installment-iban" class="label">IBAN des Zahlers <span class="text-muted-foreground">(optional)</span></label>
        <input id="installment-iban" v-model="form.payerIban" type="text" class="input" placeholder="DE…" />
        <p class="mt-1.5 text-xs text-muted-foreground">
          Künftige Überweisungen von dieser IBAN werden als Rate vorgeschlagen.
        </p>
      </div>
      <div>
        <label for="installment-contract" class="label">Unterschriebener Vertrag <span class="text-muted-foreground">(PDF)</span></label>
        <input id="installment-contract" type="file" accept="application/pdf" class="input py-2"
          @change="pickContract" />
        <p class="mt-1.5 text-xs text-muted-foreground">
          {{ plan?.contract ? 'Ersetzt den hinterlegten Vertrag.' : 'Kann auch später auf der Detailseite nachgereicht werden.' }}
        </p>
      </div>
      <div>
        <label for="installment-note" class="label">Notiz</label>
        <textarea id="installment-note" v-model="form.note" rows="3" class="input"
          placeholder="Vereinbarung, Ansprechpartner …" />
      </div>

      <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="submit" form="installment-plan-form" class="btn btn-primary" :disabled="isSaving">
        <Loader2 v-if="isSaving" class="animate-spin" />
        {{ assignsTransaction ? 'Anlegen & zuordnen' : 'Speichern' }}
      </button>
    </template>
  </AppSheet>
</template>
