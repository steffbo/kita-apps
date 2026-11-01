<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { CornerDownRight, Download, Loader2, MessageSquareWarning } from 'lucide-vue-next';
import { api } from '@/api';
import type { OwnDonationYear, OwnEinstufungLetter, OwnFees } from '@/api/types';
import { formatCurrency, formatDate, formatMonthName, todayISO } from '@/utils/format';
import { getFeePeriodLabel, getFeeTypeName } from '@/utils/fees';
import ReportDialog from '@/components/family/ReportDialog.vue';
import DonationYearList from '@/components/DonationYearList.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import { currentYear, yearOptions } from '@/utils/years';
type FeeRow = OwnFees['items'][number];
const year = ref(currentYear());
const fees = ref<OwnFees | null>(null);
const error = ref('');
const reportId = ref<string | null>(null);
const onlyOpen = ref(false);
const years = yearOptions(year.value);
const status: Record<string, string> = { OPEN: 'Offen', PAID: 'Bezahlt', OVERDUE: 'Überfällig' };
function tone(value: string) {
  return value === 'PAID' ? 'badge-success'
    : value === 'OVERDUE' ? 'badge-danger' : 'badge-warning';
}
// A Mahngebühr is listed right below its fee; one without its fee in the list names it instead.
const groups = computed(() => {
  const items = fees.value?.items ?? [];
  const ids = new Set(items.map(f => f.id));
  const reminders = new Map<string, FeeRow[]>();
  for (const f of items) {
    if (f.reminderForId && ids.has(f.reminderForId)) {
      reminders.set(f.reminderForId, [...(reminders.get(f.reminderForId) ?? []), f]);
    }
  }
  return items.filter(f => !f.reminderForId || !ids.has(f.reminderForId))
    .map(fee => ({ fee, reminders: reminders.get(fee.id) ?? [] }))
    .filter(g => !onlyOpen.value || [g.fee, ...g.reminders].some(f => f.status !== 'PAID'));
});
const showChild = computed(() => new Set(fees.value?.items.map(f => f.childId)).size > 1);
function period(fee: FeeRow) {
  return getFeePeriodLabel({ year: fee.year, month: fee.month, reminderFor: fee.baseFeeType
    ? { feeType: fee.baseFeeType, year: fee.baseYear ?? fee.year, month: fee.baseMonth } : undefined });
}
// Days until the due date (negative when overdue), both as Berlin calendar dates.
function daysUntil(dueDate: string) {
  const day = (iso: string) => Date.UTC(+iso.slice(0, 4), +iso.slice(5, 7) - 1, +iso.slice(8, 10));
  return Math.round((day(dueDate.slice(0, 10)) - day(todayISO())) / 86_400_000);
}
function deadline(dueDate: string) {
  const days = daysUntil(dueDate);
  if (days > 1) return `noch ${days} Tage`;
  if (days === 1) return 'morgen fällig';
  if (days === 0) return 'heute fällig';
  return days === -1 ? 'seit 1 Tag überfällig' : `seit ${-days} Tagen überfällig`;
}
async function load() {
  error.value = '';
  try { fees.value = await api.getOwnFees(year.value); }
  catch (e) { error.value = e instanceof Error ? e.message : 'Beiträge konnten nicht geladen werden'; }
}
onMounted(load); watch(year, load);

// Einstufung letters as emailed; independent of the selected year.
const letters = ref<OwnEinstufungLetter[]>([]);
const lettersError = ref('');
const downloading = ref<string | null>(null);
async function loadLetters() {
  try { letters.value = await api.getOwnEinstufungLetters(); }
  catch (e) { lettersError.value = e instanceof Error ? e.message : 'Einstufungsbriefe konnten nicht geladen werden'; }
}
async function downloadLetter(letter: OwnEinstufungLetter) {
  downloading.value = letter.id;
  lettersError.value = '';
  try {
    const file = await api.getOwnEinstufungLetter(letter.id);
    const url = URL.createObjectURL(file.blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = file.filename ?? letter.filename;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (e) {
    lettersError.value = e instanceof Error ? e.message : 'PDF konnte nicht geladen werden';
  } finally {
    downloading.value = null;
  }
}
onMounted(loadLetters);

// Donations credited to the household; independent of the selected year, hidden when there are none.
const donations = ref<OwnDonationYear[]>([]);
const donationsError = ref('');
onMounted(async () => {
  try { donations.value = await api.getOwnDonations(); }
  catch (e) { donationsError.value = e instanceof Error ? e.message : 'Spenden konnten nicht geladen werden'; }
});
</script>
<template>
  <div class="space-y-5">
    <PageHeader title="Beiträge" />
    <label class="label block max-w-40">Jahr
      <select v-model.number="year" class="input">
        <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
      </select>
    </label>
    <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    <div v-if="fees" class="grid gap-3 sm:grid-cols-2">
      <button type="button" :aria-pressed="onlyOpen" title="Nur offene Beiträge anzeigen"
        class="btn btn-secondary h-auto min-h-14 w-full justify-start p-4 text-left"
        :class="onlyOpen ? 'border-primary ring-2 ring-primary' : ''" @click="onlyOpen = !onlyOpen">
        Offen: <strong>{{ formatCurrency(fees.openTotal) }}</strong>
      </button>
      <div class="card p-4">Bezahlt: <strong>{{ formatCurrency(fees.paidTotal) }}</strong></div>
    </div>
    <div v-if="onlyOpen" class="flex flex-wrap items-center gap-3 text-sm">
      <span class="text-muted-foreground">Nur offene Beiträge</span>
      <button type="button" class="btn btn-secondary"
        @click="onlyOpen = false">Alle anzeigen</button>
    </div>
    <EmptyState v-if="fees && !groups.length"
      :text="onlyOpen ? 'Keine offenen Beiträge.' : 'Für dieses Jahr gibt es keine Beiträge.'" />
    <div v-else-if="fees">
      <ul class="card divide-y sm:hidden">
        <li v-for="fee in groups.flatMap(g => [g.fee, ...g.reminders])" :key="fee.id" class="flex items-start gap-3 p-4">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2"><span class="badge">{{ getFeeTypeName(fee.feeType) }}</span><span class="badge" :class="tone(fee.status)">{{ status[fee.status] ?? fee.status }}</span></div>
            <p class="mt-2 font-medium">{{ period(fee) }}<span v-if="showChild"> · {{ fee.childName }}</span></p>
            <p class="text-[13px] text-muted-foreground">Fällig {{ formatDate(fee.dueDate) }} · {{ fee.paidAt ? `bezahlt am ${formatDate(fee.paidAt)}` : deadline(fee.dueDate) }}</p>
            <p class="mt-1 font-semibold">{{ formatCurrency(fee.amount) }}<span v-if="fee.status !== 'PAID' && fee.paidAmount > 0" class="ml-2 text-xs font-normal text-muted-foreground">bezahlt {{ formatCurrency(fee.paidAmount) }}</span></p>
          </div>
          <button type="button" title="Fehler melden"
            :aria-label="`Fehler melden: ${getFeeTypeName(fee.feeType)} ${fee.month ? formatMonthName(fee.month) + ' ' : ''}${fee.year}`"
            class="btn btn-secondary btn-icon btn-sm" @click="reportId = fee.id"><MessageSquareWarning /></button>
        </li>
      </ul>
      <div class="hidden overflow-x-auto card sm:block">
      <table class="table w-full text-sm">
        <thead class="border-b text-left text-muted-foreground">
          <tr>
            <th class="px-4 py-2 font-medium">Art</th>
            <th class="px-4 py-2 font-medium">Zeitraum</th>
            <th v-if="showChild" class="px-4 py-2 font-medium">Kind</th>
            <th class="px-4 py-2 text-right font-medium">Betrag</th>
            <th class="px-4 py-2 font-medium">Fällig</th>
            <th class="px-4 py-2 font-medium">Status</th>
            <th class="px-2 py-2"><span class="sr-only">Aktionen</span></th>
          </tr>
        </thead>
        <tbody>
          <!-- Stripes per fee with its reminders, so the global zebra rule does not split them. -->
          <template v-for="(g, gi) in groups" :key="g.fee.id">
            <tr v-for="(fee, i) in [g.fee, ...g.reminders]" :key="fee.id"
              :class="gi % 2 ? 'bg-muted/45' : 'bg-card'">
              <td class="whitespace-nowrap px-4 py-2">
                <CornerDownRight v-if="i > 0" class="mr-1 inline h-4 w-4 text-muted-foreground" />
                <span class="badge">{{ getFeeTypeName(fee.feeType) }}</span>
              </td>
              <td class="whitespace-nowrap px-4 py-2">{{ i > 0 ? '' : period(fee) }}</td>
              <td v-if="showChild" class="whitespace-nowrap px-4 py-2">{{ fee.childName }}</td>
              <td class="whitespace-nowrap px-4 py-2 text-right font-semibold">
                {{ formatCurrency(fee.amount) }}
                <span v-if="fee.status !== 'PAID' && fee.paidAmount > 0"
                  class="block text-xs font-normal text-muted-foreground">
                  bezahlt {{ formatCurrency(fee.paidAmount) }}</span>
              </td>
              <td class="whitespace-nowrap px-4 py-2">{{ formatDate(fee.dueDate) }}</td>
              <td class="whitespace-nowrap px-4 py-2">
                <span class="badge" :class="tone(fee.status)">
                  {{ status[fee.status] ?? fee.status }}</span>
                <span v-if="fee.paidAt" class="ml-2 text-xs text-muted-foreground">
                  am {{ formatDate(fee.paidAt) }}</span>
                <span v-else-if="fee.status !== 'PAID'" class="ml-2 text-xs"
                  :class="daysUntil(fee.dueDate) < 0 ? 'text-destructive' : 'text-muted-foreground'">
                  {{ deadline(fee.dueDate) }}</span>
              </td>
              <td class="px-2 py-2 text-right">
                <button type="button" title="Fehler melden"
                  :aria-label="`Fehler melden: ${getFeeTypeName(fee.feeType)} ${fee.month
                    ? formatMonthName(fee.month) + ' ' : ''}${fee.year}`"
                  class="btn btn-secondary btn-icon btn-sm"
                  @click="reportId = fee.id"><MessageSquareWarning class="h-4 w-4" /></button>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
      </div>
    </div>
    <section class="space-y-2" aria-labelledby="letters-heading">
      <h2 id="letters-heading" class="text-lg font-semibold">Einstufungsbriefe</h2>
      <p v-if="lettersError" role="alert" class="note note-danger">{{ lettersError }}</p>
      <p v-if="!letters.length" class="text-sm text-muted-foreground">
        Hier erscheinen die Einstufungen, die wir euch per E-Mail schicken.</p>
      <ul v-else class="divide-y card">
        <li v-for="letter in letters" :key="letter.id"
          class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
          <div>
            <p class="font-medium">{{ letter.subject }}</p>
            <p class="text-muted-foreground">{{ letter.childFirstName }} · per E-Mail am {{ formatDate(letter.sentAt) }}</p>
          </div>
          <button type="button" :disabled="downloading !== null"
            class="btn btn-secondary btn-sm"
            :aria-label="`${letter.filename} herunterladen`" @click="downloadLetter(letter)">
            <Loader2 v-if="downloading === letter.id" class="h-4 w-4 animate-spin" />
            <Download v-else class="h-4 w-4" />
            PDF
          </button>
        </li>
      </ul>
    </section>
    <p v-if="donationsError" role="alert" class="note note-danger">{{ donationsError }}</p>
    <section v-if="donations.length" class="space-y-2" aria-labelledby="donations-heading">
      <h2 id="donations-heading" class="text-lg font-semibold">Spenden</h2>
      <DonationYearList :years="donations" />
    </section>
    <ReportDialog v-if="reportId" topic="FEE" :reference-id="reportId"
      @close="reportId = null" @saved="reportId = null" />
  </div>
</template>
