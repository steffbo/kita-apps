<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import { api } from '@/api';
import type { InstallmentDetail } from '@/api/types';
import { AlertTriangle, HandCoins, Loader2, Plus } from 'lucide-vue-next';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import InstallmentPlanForm from '@/components/installments/InstallmentPlanForm.vue';
import { formatCurrency } from '@/utils/format';
import {
  PLAN_FILTERS, type PlanFilter, debtorKindLabel, intervalLabel, matchesPlanFilter, nextDueText, paidRateCount,
  planStateBadgeClass, planStateLabel,
} from '@/utils/installments';

const router = useRouter();
const plans = ref<InstallmentDetail[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);
const filter = ref<PlanFilter>('active');
const showForm = ref(false);

async function load(): Promise<void> {
  isLoading.value = true;
  error.value = null;
  try {
    plans.value = await api.getInstallmentPlans();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ratenzahlungen konnten nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(load);

// All plans are loaded once; chips filter on the client.
const filterOptions = computed(() => PLAN_FILTERS.map(option => ({
  ...option,
  count: plans.value.filter(plan => matchesPlanFilter(plan, option.value)).length,
})));
const visiblePlans = computed(() => plans.value.filter(plan => matchesPlanFilter(plan, filter.value)));

const openPlans = computed(() => plans.value.filter(plan => matchesPlanFilter(plan, 'active')));
const overduePlans = computed(() => plans.value.filter(plan => matchesPlanFilter(plan, 'overdue')));
const restTotal = computed(() => openPlans.value.reduce((sum, plan) => sum + plan.summary.rest, 0));
const overdueTotal = computed(() => overduePlans.value.reduce((sum, plan) => sum + plan.summary.overdue, 0));

function openPlan(planId: string): void {
  void router.push(`/ratenzahlungen/${planId}`);
}
</script>

<template>
  <div>
    <PageHeader title="Ratenzahlungen">
      <template #actions>
        <button type="button" class="btn btn-primary" @click="showForm = true">
          <Plus />Ratenzahlung anlegen
        </button>
      </template>
    </PageHeader>

    <div class="mb-4 grid grid-cols-2 gap-3">
      <div class="card p-4">
        <p class="text-[13px] text-muted-foreground">Restschuld gesamt</p>
        <p class="mt-1 text-xl font-semibold tabular-nums">{{ formatCurrency(restTotal) }}</p>
      </div>
      <div class="card p-4">
        <p class="text-[13px] text-muted-foreground">Davon überfällig</p>
        <p class="mt-1 text-xl font-semibold tabular-nums" :class="overdueTotal > 0 && 'text-destructive'">
          {{ formatCurrency(overdueTotal) }}
        </p>
      </div>
    </div>

    <div class="-mx-4 mb-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:flex-wrap sm:px-0" role="group" aria-label="Filter">
      <button v-for="option in filterOptions" :key="option.value" type="button" class="chip"
        :aria-pressed="filter === option.value" @click="filter = option.value">
        {{ option.label }}<span class="chip-count">{{ option.count }}</span>
      </button>
    </div>

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
    </div>

    <div v-else-if="error" class="note note-danger" role="alert">
      <AlertTriangle />
      <div class="flex-1">{{ error }}</div>
      <button type="button" class="font-medium underline" @click="load">Erneut versuchen</button>
    </div>

    <div v-else class="card overflow-hidden">
      <EmptyState v-if="visiblePlans.length === 0" :icon="HandCoins" text="Keine Ratenzahlungen">
        <button type="button" class="btn btn-secondary btn-sm" @click="showForm = true">
          <Plus />Ratenzahlung anlegen
        </button>
      </EmptyState>

      <ul v-else class="divide-y">
        <li v-for="item in visiblePlans" :key="item.plan.id">
          <RouterLink :to="`/ratenzahlungen/${item.plan.id}`"
            class="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-accent">
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium">{{ item.plan.debtorName }}</span>
              <span class="block text-[13px] text-muted-foreground">
                {{ debtorKindLabel(item.plan) }} · {{ formatCurrency(item.plan.installmentAmount) }}
                {{ intervalLabel(item.plan.intervalMonths) }}
              </span>
              <span class="block text-[13px] tabular-nums text-muted-foreground">
                {{ paidRateCount(item) }}/{{ item.schedule.length }} Raten ·
                {{ nextDueText(item) }}
              </span>
              <span v-if="!item.contract" class="block text-[13px] text-warning">Vertrag fehlt</span>
            </span>
            <span class="flex shrink-0 flex-col items-end gap-1.5">
              <span class="font-semibold tabular-nums">{{ formatCurrency(item.summary.rest) }}</span>
              <span :class="planStateBadgeClass(item.summary.state)">{{ planStateLabel(item.summary.state) }}</span>
            </span>
          </RouterLink>
        </li>
      </ul>
    </div>

    <InstallmentPlanForm v-if="showForm" @close="showForm = false" @saved="openPlan" />
  </div>
</template>
