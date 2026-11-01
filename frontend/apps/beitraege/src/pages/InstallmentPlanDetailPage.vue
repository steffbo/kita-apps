<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { api } from '@/api';
import type { InstallmentDetail, InstallmentRate } from '@/api/types';
import { AlertTriangle, Banknote, CalendarClock, Check, Loader2, Mail, Pencil, Trash2 } from 'lucide-vue-next';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import DetailRow from '@/components/ui/DetailRow.vue';
import InstallmentContractCard from '@/components/installments/InstallmentContractCard.vue';
import InstallmentPlanForm from '@/components/installments/InstallmentPlanForm.vue';
import InstallmentPaymentSheet from '@/components/installments/InstallmentPaymentSheet.vue';
import InstallmentFinalSheet from '@/components/installments/InstallmentFinalSheet.vue';
import InstallmentCloseSheet from '@/components/installments/InstallmentCloseSheet.vue';
import InstallmentReminderSheet from '@/components/installments/InstallmentReminderSheet.vue';
import { formatCurrency, formatDate } from '@/utils/format';
import { formatEmailType } from '@/utils/reminders';
import {
  debtorKindLabel, intervalLabel, nextReminderLabel, overdueRates, paidRateCount, planStateBadgeClass, planStateLabel,
  rateStatusBadgeClass, rateStatusLabel,
} from '@/utils/installments';

type Sheet = 'edit' | 'payment' | 'final' | 'close' | 'remind';

const route = useRoute();
const planId = computed(() => String(route.params.id));
const detail = ref<InstallmentDetail | null>(null);
const isLoading = ref(true);
const loadError = ref<string | null>(null);
const actionError = ref<string | null>(null);
const sheet = ref<Sheet | null>(null);
const deletingPaymentId = ref<string | null>(null);

async function load(): Promise<void> {
  loadError.value = null;
  try {
    detail.value = await api.getInstallmentPlan(planId.value);
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Ratenzahlung konnte nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
}

// Dialogs close and the detail reloads after every change.
function reload(): void {
  sheet.value = null;
  deletingPaymentId.value = null;
  void load();
}

async function run(task: () => Promise<void>): Promise<void> {
  actionError.value = null;
  try {
    await task();
    reload();
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Aktion fehlgeschlagen';
  }
}

onMounted(load);
watch(planId, () => {
  isLoading.value = true;
  void load();
});

const isOpen = computed(() => detail.value?.plan.status === 'ACTIVE');
const progress = computed(() => {
  const total = detail.value?.summary.effectiveTotal ?? 0;
  return total > 0 ? Math.min(100, Math.round(((detail.value?.summary.paid ?? 0) / total) * 100)) : 0;
});
const lastRate = computed(() => {
  const schedule = detail.value?.schedule ?? [];
  return schedule[schedule.length - 1];
});
const sortedPayments = computed(() => [...(detail.value?.payments ?? [])]
  .sort((a, b) => b.paidOn.localeCompare(a.paidOn)));

// The Schlussrate is always the last rate of the schedule.
function rateLabel(rate: InstallmentRate, index: number): string {
  const isFinal = !!detail.value?.plan.finalDueDate && index === (detail.value?.schedule.length ?? 0) - 1;
  return isFinal ? 'Schlussrate' : `Rate ${rate.number}`;
}

function reminderCharge(emailLogId: string): number | undefined {
  return detail.value?.charges.find(charge => charge.emailLogId === emailLogId)?.amount;
}

function clearFinalDueDate(): void {
  void run(() => api.clearInstallmentFinalDueDate(planId.value));
}

function deletePayment(paymentId: string): void {
  deletingPaymentId.value = null;
  void run(() => api.deleteInstallmentPayment(planId.value, paymentId));
}
</script>

<template>
  <div>
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
    </div>

    <div v-else-if="loadError || !detail" class="note note-danger" role="alert">
      <AlertTriangle />{{ loadError ?? 'Ratenzahlung nicht gefunden' }}
    </div>

    <template v-else>
      <PageHeader :title="detail.plan.debtorName" back="/ratenzahlungen">
        <template #actions>
          <button type="button" class="btn btn-secondary" @click="sheet = 'edit'">
            <Pencil />Bearbeiten
          </button>
        </template>
      </PageHeader>

      <div v-if="actionError" class="note note-danger mb-4" role="alert"><AlertTriangle />{{ actionError }}</div>

      <div v-if="isOpen && overdueRates(detail).length" class="note note-danger mb-4">
        <AlertTriangle />
        <span class="flex-1">
          {{ overdueRates(detail).length }} {{ overdueRates(detail).length === 1 ? 'Rate' : 'Raten' }} überfällig ·
          {{ formatCurrency(detail.summary.overdue) }}
        </span>
        <button v-if="detail.reminderDue" type="button" class="btn btn-secondary btn-sm" @click="sheet = 'remind'">
          {{ nextReminderLabel(detail) }} senden
        </button>
      </div>
      <div v-if="!isOpen" class="note note-info mb-4">
        <span>
          {{ planStateLabel(detail.plan.status) }}<template v-if="detail.plan.closedAt">
            am {{ formatDate(detail.plan.closedAt) }}</template>{{ detail.plan.closeReason ? `: ${detail.plan.closeReason}` : '' }}
        </span>
      </div>

      <div class="card mb-6 p-5">
        <p class="text-[13px] text-muted-foreground">Restschuld</p>
        <p class="mt-1 text-2xl font-semibold tabular-nums">
          {{ formatCurrency(detail.summary.rest) }}
          <span class="text-sm font-normal text-muted-foreground">von {{ formatCurrency(detail.summary.effectiveTotal) }}</span>
        </p>
        <div class="mt-3 h-2 overflow-hidden rounded-full bg-muted" role="progressbar" :aria-valuenow="progress"
          aria-valuemin="0" aria-valuemax="100" aria-label="Bezahlt">
          <div class="h-full rounded-full bg-primary" :style="{ width: `${progress}%` }" />
        </div>
        <div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
          <span class="tabular-nums">
            {{ formatCurrency(detail.summary.paid) }} bezahlt · {{ paidRateCount(detail) }}/{{ detail.schedule.length }} Raten
          </span>
          <span :class="planStateBadgeClass(detail.summary.state)">{{ planStateLabel(detail.summary.state) }}</span>
        </div>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <div class="space-y-6">
          <section>
            <div class="mb-2 flex min-h-8 items-center justify-between gap-2">
              <h2 class="text-base font-semibold">Vereinbarung</h2>
              <template v-if="isOpen && detail.summary.rest > 0">
                <button v-if="detail.plan.finalDueDate" type="button" class="btn btn-secondary btn-sm"
                  @click="clearFinalDueDate">Schlussrate aufheben</button>
                <button v-else type="button" class="btn btn-secondary btn-sm" @click="sheet = 'final'">
                  <CalendarClock />Schlussrate
                </button>
              </template>
            </div>
            <dl class="card divide-y text-sm">
              <DetailRow label="Ratenhöhe" value-class="tabular-nums">{{ formatCurrency(detail.plan.installmentAmount) }}</DetailRow>
              <DetailRow label="Intervall">{{ intervalLabel(detail.plan.intervalMonths) }}</DetailRow>
              <DetailRow label="Erste Rate">{{ formatDate(detail.plan.firstDueDate) }}</DetailRow>
              <DetailRow :label="detail.plan.finalDueDate ? 'Schlussrate' : 'Letzte Rate'" value-class="tabular-nums">
                  {{ formatDate(lastRate?.dueDate) }}<template v-if="detail.plan.finalDueDate && lastRate">
                    · {{ formatCurrency(lastRate.amount) }}</template>
                </DetailRow>
              <DetailRow label="Bezug" value-class="text-right">{{ detail.plan.reference || '—' }}</DetailRow>
              <DetailRow label="IBAN für Abgleich" value-class="font-mono text-xs">{{ detail.plan.payerIban || '—' }}</DetailRow>
            </dl>
          </section>

          <section>
            <h2 class="mb-2 text-base font-semibold">Vertrag</h2>
            <InstallmentContractCard :detail="detail" @changed="reload" />
          </section>

          <section>
            <h2 class="mb-2 text-base font-semibold">Schuldner</h2>
            <dl class="card divide-y text-sm">
              <DetailRow :label="detail.plan.householdId ? 'Haushalt' : 'Name'" value-class="text-right">{{ detail.plan.debtorName }}</DetailRow>
              <DetailRow v-if="detail.plan.debtorAddress" label="Anschrift" value-class="whitespace-pre-line text-right">{{ detail.plan.debtorAddress }}</DetailRow>
              <DetailRow label="Art">{{ debtorKindLabel(detail.plan) }}</DetailRow>
            </dl>
          </section>

          <section>
            <div class="mb-2 flex min-h-8 items-center justify-between gap-2">
              <h2 class="text-base font-semibold">E-Mail für Erinnerungen</h2>
              <button type="button" class="btn btn-secondary btn-sm" @click="sheet = 'edit'">
                <Pencil />Ändern
              </button>
            </div>
            <div v-if="detail.plan.emails.length" class="card flex flex-wrap gap-1.5 p-3">
              <span v-for="email in detail.plan.emails" :key="email" class="badge h-7 px-2.5">
                <Mail />{{ email }}
              </span>
            </div>
            <div v-else class="note note-warning">
              <AlertTriangle />
              <span>Keine Adresse hinterlegt – Erinnerungen können nicht versendet werden.</span>
            </div>
          </section>

          <section v-if="detail.plan.note">
            <h2 class="mb-2 text-base font-semibold">Notiz</h2>
            <div class="card whitespace-pre-line p-4 text-sm">{{ detail.plan.note }}</div>
          </section>
        </div>

        <div class="space-y-6">
          <section>
            <h2 class="mb-2 text-base font-semibold">Ratenplan</h2>
            <ul class="card divide-y">
              <li v-for="(rate, index) in detail.schedule" :key="rate.number"
                class="flex min-h-14 items-center gap-3 px-4 py-2.5">
                <span class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold tabular-nums">
                  <Check v-if="rate.status === 'PAID'" class="h-4 w-4 text-success" />
                  <template v-else>{{ rate.number }}</template>
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block font-medium tabular-nums">{{ rateLabel(rate, index) }} · {{ formatCurrency(rate.amount) }}</span>
                  <span class="block text-xs text-muted-foreground">
                    fällig {{ formatDate(rate.dueDate) }}<template v-if="rate.paid > 0 && rate.status !== 'PAID'">
                      · {{ formatCurrency(rate.paid) }} bezahlt</template>
                  </span>
                </span>
                <span :class="rateStatusBadgeClass(rate.status)">{{ rateStatusLabel(rate.status) }}</span>
              </li>
            </ul>
          </section>

          <section>
            <div class="mb-2 flex min-h-8 items-center justify-between gap-2">
              <h2 class="text-base font-semibold">Zahlungen</h2>
              <button v-if="isOpen" type="button" class="btn btn-secondary btn-sm" @click="sheet = 'payment'">
                <Banknote />Zahlung
              </button>
            </div>
            <div class="card">
              <EmptyState v-if="detail.payments.length === 0" :icon="Banknote" text="Noch keine Zahlung" />
              <ul v-else class="divide-y">
                <li v-for="payment in sortedPayments" :key="payment.id" class="flex min-h-14 items-center gap-3 px-4 py-2.5">
                  <span class="min-w-0 flex-1">
                    <span class="block font-medium">{{ formatDate(payment.paidOn) }}</span>
                    <span class="block truncate text-xs text-muted-foreground">
                      {{ payment.source === 'BANK' ? 'Bank · ' : '' }}{{ payment.note || '—' }}
                    </span>
                  </span>
                  <span class="font-medium tabular-nums text-success">+{{ formatCurrency(payment.amount) }}</span>
                  <template v-if="isOpen">
                    <template v-if="deletingPaymentId === payment.id">
                      <button type="button" class="btn btn-danger btn-sm" @click="deletePayment(payment.id)">Ja</button>
                      <button type="button" class="btn btn-secondary btn-sm" @click="deletingPaymentId = null">Nein</button>
                    </template>
                    <button v-else type="button" class="btn btn-ghost btn-sm" aria-label="Zahlung löschen"
                      @click="deletingPaymentId = payment.id"><Trash2 /></button>
                  </template>
                </li>
              </ul>
            </div>
          </section>

          <section>
            <h2 class="mb-2 text-base font-semibold">Erinnerungsverlauf</h2>
            <div class="card">
              <EmptyState v-if="detail.reminders.length === 0" :icon="Mail" text="Noch nichts versendet" />
              <ul v-else class="divide-y">
                <li v-for="mail in detail.reminders" :key="mail.id" class="flex min-h-14 items-center gap-3 px-4 py-2.5">
                  <Mail class="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span class="min-w-0 flex-1">
                    <span class="block font-medium">
                      {{ formatEmailType(mail.emailType) }}<template v-if="reminderCharge(mail.id)">
                        + Mahngebühr {{ formatCurrency(reminderCharge(mail.id) ?? 0) }}</template>
                    </span>
                    <span class="block truncate text-xs text-muted-foreground">{{ formatDate(mail.sentAt) }} · {{ mail.toEmail }}</span>
                  </span>
                </li>
              </ul>
            </div>
          </section>

          <button v-if="isOpen" type="button" class="btn btn-danger-outline self-start" @click="sheet = 'close'">
            Ratenzahlung beenden …
          </button>
        </div>
      </div>

      <InstallmentPlanForm v-if="sheet === 'edit'" :plan="detail" @close="sheet = null" @saved="reload" />
      <InstallmentPaymentSheet v-if="sheet === 'payment'" :detail="detail" @close="sheet = null" @saved="reload" />
      <InstallmentFinalSheet v-if="sheet === 'final'" :detail="detail" @close="sheet = null" @saved="reload" />
      <InstallmentCloseSheet v-if="sheet === 'close'" :detail="detail" @close="sheet = null" @saved="reload" />
      <InstallmentReminderSheet v-if="sheet === 'remind'" :detail="detail" @close="sheet = null" @sent="reload"
        @edit="sheet = 'edit'" />
    </template>
  </div>
</template>
