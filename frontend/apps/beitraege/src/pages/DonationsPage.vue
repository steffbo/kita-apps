<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction, DonationReport } from '@/api/types';
import { AlertTriangle, Heart, Loader2, Pencil } from 'lucide-vue-next';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import DonationAssignPanel from '@/components/import/DonationAssignPanel.vue';
import { formatCurrency, formatDate } from '@/utils/format';

const report = ref<DonationReport | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  isLoading.value = true;
  error.value = null;
  try {
    report.value = await api.getDonations();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Spenden konnten nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
}

const editing = ref<BankTransaction | null>(null);

async function saved(): Promise<void> {
  editing.value = null;
  await load();
}

onMounted(load);
</script>

<template>
  <div>
    <PageHeader title="Spenden" />

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
    </div>

    <div v-else-if="error" class="note note-danger" role="alert">
      <AlertTriangle />
      <div class="flex-1">{{ error }}</div>
      <button type="button" class="font-medium underline" @click="load">Erneut versuchen</button>
    </div>

    <div v-else-if="!report || report.count === 0" class="card">
      <EmptyState :icon="Heart" text="Noch keine Spenden zugeordnet. Zuordnen im Bankabgleich über „Spende“." />
    </div>

    <template v-else>
      <div class="card mb-4 p-4">
        <p class="text-[13px] text-muted-foreground">Spenden gesamt · {{ report.count }}</p>
        <p class="mt-1 text-xl font-semibold tabular-nums" data-testid="donations-total">
          {{ formatCurrency(report.total) }}
        </p>
      </div>

      <section v-for="year in report.years" :key="year.year" class="mb-6" :aria-label="`Spenden ${year.year}`">
        <div class="mb-2 flex items-baseline justify-between gap-3">
          <h2 class="text-base font-semibold">{{ year.year }}</h2>
          <span class="text-sm tabular-nums text-muted-foreground">
            {{ year.count }} · <span class="font-semibold text-foreground">{{ formatCurrency(year.total) }}</span>
          </span>
        </div>

        <div class="card overflow-hidden">
          <ul class="divide-y sm:hidden">
            <li v-for="tx in year.transactions" :key="tx.id">
              <button type="button" class="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-accent"
                :aria-label="`Spende von ${tx.payerName || 'Unbekannt'} bearbeiten`" @click="editing = tx">
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-medium">{{ tx.payerName || 'Unbekannt' }}</span>
                  <span class="block text-[13px] text-muted-foreground">
                    {{ formatDate(tx.bookingDate) }}<template v-if="tx.assignment?.householdName"> · {{ tx.assignment.householdName }}</template><template v-if="tx.assignment?.note"> · {{ tx.assignment.note }}</template>
                  </span>
                </span>
                <span class="shrink-0 font-semibold tabular-nums">{{ formatCurrency(tx.amount) }}</span>
              </button>
            </li>
          </ul>
          <table class="table hidden sm:table">
            <thead>
              <tr>
                <th>Datum</th><th>Zahler</th><th>Haushalt</th><th>Verwendungszweck</th><th>Notiz</th><th class="text-right">Betrag</th><th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="tx in year.transactions" :key="tx.id">
                <td class="whitespace-nowrap">{{ formatDate(tx.bookingDate) }}</td>
                <td>{{ tx.payerName || 'Unbekannt' }}</td>
                <td>{{ tx.assignment?.householdName }}</td>
                <td class="text-muted-foreground">{{ tx.description }}</td>
                <td>{{ tx.assignment?.note }}</td>
                <td class="whitespace-nowrap text-right font-semibold tabular-nums">{{ formatCurrency(tx.amount) }}</td>
                <td class="text-right">
                  <button type="button" class="btn btn-ghost btn-icon btn-sm"
                    :aria-label="`Spende von ${tx.payerName || 'Unbekannt'} bearbeiten`" @click="editing = tx">
                    <Pencil />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <AppSheet v-if="editing" title="Spende bearbeiten" @close="editing = null">
      <DonationAssignPanel :transaction="editing" @matched="saved" />
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="editing = null">Abbrechen</button>
      </template>
    </AppSheet>
  </div>
</template>
