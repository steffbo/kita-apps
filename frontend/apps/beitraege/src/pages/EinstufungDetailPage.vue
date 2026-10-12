<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type {
  Einstufung,
  EinstufungPreview,
  Child,
  Household,
  IncomeBasis,
  IncomeDetails,
  HouseholdIncomeCalculation,
  ChildcareExpectationSyncResult,
  PreviewEinstufungRequest,
} from '@/api/types';
import {
  AlertTriangle,
  ArrowLeft,
  Calculator,
  Loader2,
  Save,
  Euro,
  Info,
  User,
  Home,
  FileText,
  CheckCircle,
  Mail,
  RotateCcw,
  Download,
  ExternalLink,
  Send,
} from 'lucide-vue-next';
import IncomeForm from '@/components/IncomeForm.vue';
import EinstufungPDF from '@/components/EinstufungPDF.vue';
import ChildSearchSelect from '@/components/ChildSearchSelect.vue';
import { formatCurrency, formatDate, formatDateTime, formatEuroAmount, formatMonthYear, todayISO } from '@/utils/format';

const route = useRoute();
const router = useRouter();

const isNew = computed(() => route.params.id === 'neu');
const einstufungId = computed(() => (isNew.value ? null : route.params.id as string));
const isFollowUpMode = computed(() => isNew.value && (route.query.followUp === '1' || !!route.query.sourceId));

const isLoading = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);
const saveSuccess = ref(false);
const expectationChanges = ref<ChildcareExpectationSyncResult | null>(null);

// Loaded einstufung (edit mode)
const einstufung = ref<Einstufung | null>(null);
const sourceEinstufung = ref<Einstufung | null>(null);
const previousEinstufung = ref<Einstufung | null>(null);
const successorEinstufung = ref<Einstufung | null>(null);

// Form data
const currentYear = new Date().getFullYear();
const selectedChildId = ref('');
const selectedYear = ref(currentYear);
const yearOptions = computed(() =>
  [...new Set([currentYear - 1, currentYear, currentYear + 1, selectedYear.value])].sort((a, b) => a - b));
const validFrom = ref(`${new Date().getFullYear()}-01-01`);
const childrenCount = ref(1);
const incomeBasis = ref<IncomeBasis>('INCOME');
const provisional = ref(false);
const notes = ref('');

// Income form
function emptyIncome(): IncomeDetails {
  return {
    grossIncome: 0,
    socialSecurityShare: 0,
    privateInsurance: 0,
    tax: 0,
    advertisingCosts: 1500,
    minijobIncome: 0,
    unemploymentBenefit: 0,
    capitalIncome: 0,
    rentalIncome: 0,
    otherIncome: 0,
    profit: 0,
    welfareExpense: 0,
    selfEmployedTax: 0,
    parentalBenefit: 0,
    parentalBenefitPlus: 0,
    maternityBenefit: 0,
    insurances: 0,
    maintenanceToPay: 0,
    maintenanceReceived: 0,
  };
}

const parent1Income = ref<IncomeDetails>(emptyIncome());
const parent2Income = ref<IncomeDetails>(emptyIncome());

// Reference data
const children = ref<Child[]>([]);
const households = ref<Household[]>([]);

// Computed: selected child's household
const selectedChild = computed(() => children.value.find((c) => c.id === selectedChildId.value));
const selectedHousehold = computed(() => {
  if (!selectedChild.value?.householdId) return null;
  return households.value.find((h) => h.id === selectedChild.value!.householdId);
});
const childMissingHousehold = computed(() => !!selectedChildId.value && !!selectedChild.value && !selectedChild.value.householdId);
const householdParents = computed(() => selectedHousehold.value?.parents ?? []);

// Defaults for a new (non follow-up) Einstufung, derived from the selected child,
// its household and the household's existing Einstufungen.
const householdChildren = computed(() => {
  const householdId = selectedChild.value?.householdId;
  return householdId ? children.value.filter((c) => c.householdId === householdId) : [];
});
const existingOpenEinstufung = ref<Einstufung | null>(null);
const prefillSource = ref<Einstufung | null>(null);
const isPrefilling = ref(false);
let prefillSequence = 0;

function defaultYearFor(child: Child): number {
  const entryYear = Number(child.entryDate.slice(0, 4));
  return entryYear > currentYear ? entryYear : currentYear;
}

// Mirrors the backend: a new Einstufung starts on 1 January of its year or in the entry month,
// a follow-up in the month given by the 14th/15th cut-off of the change date.
const periodStartMonth = computed(() => {
  if (!isNew.value && einstufung.value) {
    return (einstufung.value.effectiveFromMonth || einstufung.value.validFrom).slice(0, 7);
  }
  if (isFollowUpMode.value) return calculateEffectiveFromMonth(validFrom.value).slice(0, 7);
  const entry = selectedChild.value?.entryDate.split('T')[0];
  const yearStart = `${selectedYear.value}-01-01`;
  return (entry && entry > yearStart ? entry : yearStart).slice(0, 7);
});

// The child completes its third year the day before the third birthday (§ 188 Abs. 2 BGB);
// childcare is free from that month on.
const feeFreeFromMonth = computed(() => {
  const birth = selectedChild.value?.birthDate.split('T')[0];
  if (!birth) return null;
  const [year, month, day] = birth.split('-').map(Number);
  const completes = new Date(Date.UTC(year + 3, month - 1, day - 1));
  return completes.toISOString().slice(0, 7);
});
const isKindergartenAtStart = computed(() =>
  !!feeFreeFromMonth.value && feeFreeFromMonth.value <= periodStartMonth.value);

const incomeBasisOptions = computed(() => [
  { value: 'INCOME' as const, label: 'Einkommen', hint: 'nach Einkommensnachweisen', disabled: false },
  { value: 'HIGHEST_RATE' as const, label: 'Höchstsatz', hint: 'freiwillig anerkannt, ohne Nachweise', disabled: false },
  {
    value: 'NOT_REQUIRED' as const,
    label: 'Nicht erforderlich',
    hint: isKindergartenAtStart.value ? 'Kindergartenkind, beitragsfrei' : 'nur für Kindergartenkinder',
    disabled: !isKindergartenAtStart.value,
  },
]);

const INCOME_BASIS_LABELS: Record<IncomeBasis, string> = {
  INCOME: 'Einkommen',
  HIGHEST_RATE: 'Höchstsatz (freiwillig)',
  NOT_REQUIRED: 'Einkommen nicht erforderlich',
};

watch(isKindergartenAtStart, (kindergarten) => {
  if (isNew.value && !kindergarten && incomeBasis.value === 'NOT_REQUIRED') incomeBasis.value = 'INCOME';
});

function hasIncomeData(e: Einstufung): boolean {
  return [e.incomeCalculation.parent1, e.incomeCalculation.parent2].some((parent) =>
    Object.entries(parent).some(([key, value]) => key !== 'advertisingCosts' && Number(value) !== 0));
}

function resetIncomeForm() {
  childrenCount.value = 1;
  incomeBasis.value = 'INCOME';
  provisional.value = false;
  parent1Income.value = emptyIncome();
  parent2Income.value = emptyIncome();
}

async function applyChildDefaults() {
  const sequence = ++prefillSequence;
  existingOpenEinstufung.value = null;
  prefillSource.value = null;
  resetIncomeForm();

  const child = selectedChild.value;
  if (!child) return;
  selectedYear.value = defaultYearFor(child);
  const household = selectedHousehold.value;
  if (!child.householdId || !household) return;

  // Every known child of the household counts; parents may have further dependants.
  childrenCount.value = Math.max(1, householdChildren.value.length, household.childrenCountForFees ?? 0);
  // Kindergarten children are free; their parents are not asked for income.
  if (isKindergartenAtStart.value) incomeBasis.value = 'NOT_REQUIRED';
  else if (household.incomeStatus === 'MAX_ACCEPTED') incomeBasis.value = 'HIGHEST_RATE';

  isPrefilling.value = true;
  try {
    const householdEinstufungen = await api.getEinstufungenForHousehold(child.householdId);
    if (sequence !== prefillSequence) return;
    existingOpenEinstufung.value =
      householdEinstufungen.find((e) => e.childId === child.id && !e.validUntil) ?? null;
    // Newest first: the latest decision with income data or an accepted highest rate.
    const source = householdEinstufungen.find((e) =>
      e.incomeBasis === 'HIGHEST_RATE' || (e.incomeBasis === 'INCOME' && hasIncomeData(e)));
    if (!source) return;
    prefillSource.value = source;
    childrenCount.value = Math.max(childrenCount.value, source.childrenCount);
    if (incomeBasis.value === 'NOT_REQUIRED') return;
    incomeBasis.value = source.incomeBasis;
    if (source.incomeBasis === 'INCOME') {
      parent1Income.value = { ...emptyIncome(), ...source.incomeCalculation.parent1 };
      parent2Income.value = { ...emptyIncome(), ...source.incomeCalculation.parent2 };
    }
  } catch (e) {
    if (sequence !== prefillSequence) return;
    error.value = e instanceof Error
      ? `Bestehende Einstufungen konnten nicht geladen werden: ${e.message}`
      : 'Bestehende Einstufungen konnten nicht geladen werden';
  } finally {
    if (sequence === prefillSequence) isPrefilling.value = false;
  }
}

function onChildSelected() {
  if (isNew.value && !isFollowUpMode.value) void applyChildDefaults();
}

// Keeps the household defaults (children count) and only drops the copied decision.
function discardPrefill() {
  if (prefillSource.value?.incomeBasis === 'HIGHEST_RATE' && incomeBasis.value === 'HIGHEST_RATE') {
    incomeBasis.value = 'INCOME';
  }
  prefillSource.value = null;
  parent1Income.value = emptyIncome();
  parent2Income.value = emptyIncome();
}

const prefillSummary = computed(() => {
  if (incomeBasis.value === 'NOT_REQUIRED') return 'Kinderzahl übernommen';
  const what = prefillSource.value?.incomeBasis === 'HIGHEST_RATE' ? 'Höchstsatz-Anerkennung' : 'Einkommensangaben';
  return `${what} und Kinderzahl übernommen`;
});

function einstufungLabel(e: Einstufung): string {
  const name = e.child ? `${e.child.firstName} ${e.child.lastName}` : 'Geschwisterkind';
  const start = (e.effectiveFromMonth || e.validFrom).slice(0, 7);
  return `${name}, gültig ab ${formatMonthYear(Number(start.slice(0, 4)), Number(start.slice(5, 7)))}`;
}
const emailRecipients = computed(() => {
  const seen = new Set<string>();
  const recipients: string[] = [];
  for (const parent of householdParents.value) {
    const email = parent.email?.trim();
    if (!email || seen.has(email)) continue;
    seen.add(email);
    recipients.push(email);
  }
  return recipients;
});

function calculateEffectiveFromMonth(changeDate: string): string {
  const [year, month, day] = changeDate.split('-').map(Number);
  if (!year || !month || !day) return changeDate;
  const date = new Date(Date.UTC(year, month - 1, 1));
  if (day >= 15) {
    date.setUTCMonth(date.getUTCMonth() + 1);
  }
  return date.toISOString().slice(0, 10);
}

const effectiveFromMonthPreview = computed(() => {
  if (einstufung.value?.effectiveFromMonth) return einstufung.value.effectiveFromMonth.split('T')[0];
  return calculateEffectiveFromMonth(validFrom.value);
});

function formatEffectiveFromMonth(value: string): string {
  return formatMonthYear(Number(value.slice(0, 4)), Number(value.slice(5, 7)));
}

// Live preview: the backend computes the unsaved form exactly like saving would.
const preview = ref<EinstufungPreview | null>(null);
const previewError = ref<string | null>(null);
const isPreviewing = ref(false);
let previewSequence = 0;

function incomeCalculation(): HouseholdIncomeCalculation {
  return { parent1: parent1Income.value, parent2: parent2Income.value };
}

const previewRequest = computed<PreviewEinstufungRequest | null>(() => {
  if (!selectedChildId.value || childMissingHousehold.value) return null;
  const form = {
    incomeCalculation: incomeCalculation(),
    incomeBasis: incomeBasis.value,
    childrenCount: Number(childrenCount.value) || 0,
    provisional: provisional.value,
  };
  if (!isNew.value) return einstufungId.value ? { ...form, einstufungId: einstufungId.value } : null;
  if (isFollowUpMode.value) {
    return sourceEinstufung.value && validFrom.value
      ? { ...form, sourceEinstufungId: sourceEinstufung.value.id, changeDate: validFrom.value }
      : null;
  }
  if (isPrefilling.value || existingOpenEinstufung.value) return null;
  return { ...form, childId: selectedChildId.value, year: selectedYear.value };
});

// Calculation inputs as last loaded or saved; the result card shows the stored
// Einstufung until the form differs from it.
const savedFormSnapshot = ref<string | null>(null);
const formSnapshot = computed(() => {
  const request = previewRequest.value;
  return request
    ? JSON.stringify([request.incomeCalculation, request.incomeBasis, request.childrenCount, request.provisional])
    : null;
});
const hasUnsavedCalculation = computed(() => isNew.value || formSnapshot.value !== savedFormSnapshot.value);

watch(
  () => (previewRequest.value && hasUnsavedCalculation.value ? JSON.stringify(previewRequest.value) : null),
  (payload, _previous, onCleanup) => {
    const sequence = ++previewSequence;
    previewError.value = null;
    if (!payload) {
      preview.value = null;
      isPreviewing.value = false;
      return;
    }
    isPreviewing.value = true;
    const timer = setTimeout(async () => {
      try {
        const result = await api.previewEinstufung(JSON.parse(payload) as PreviewEinstufungRequest);
        if (sequence !== previewSequence) return;
        preview.value = result;
      } catch (e) {
        if (sequence !== previewSequence) return;
        preview.value = null;
        previewError.value = e instanceof Error ? e.message : 'Vorschau konnte nicht berechnet werden';
      } finally {
        if (sequence === previewSequence) isPreviewing.value = false;
      }
    }, 400);
    onCleanup(() => clearTimeout(timer));
  },
  { immediate: true },
);

const shownResult = computed(() =>
  hasUnsavedCalculation.value ? preview.value?.einstufung ?? null : einstufung.value);

// Saving needs a confirmation while the current form produces warnings.
async function confirmPreviewWarnings(): Promise<boolean> {
  const request = previewRequest.value;
  if (!request) return true;
  const result = await api.previewEinstufung(request);
  if (!result.warnings.length) return true;
  return confirm(`${result.warnings.join('\n\n')}\n\nTrotzdem speichern?`);
}

const BANK_DETAILS = `Knirpsenstadt e.V.
IBAN: DE33 3702 0500 0003 3214 00
BIC: BFSWDE33XXX
Bank für Sozialwirtschaft AG`;

const emailSubject = ref('');
const emailBody = ref('');
const emailSubjectDirty = ref(false);
const emailBodyDirty = ref(false);
const mailError = ref<string | null>(null);
const mailSuccess = ref<string | null>(null);
const isSending = ref(false);
const letterAction = ref<'view' | 'download' | null>(null);

const einstufungChild = computed(() => einstufung.value?.child ?? selectedChild.value ?? null);
const childMemberNumber = computed(() => einstufungChild.value?.memberNumber || '—');
const einstufungForPdf = computed(() => successorEinstufung.value ?? einstufung.value);
// Letter and email always describe the latest saved Einstufung of this period.
const letterBlockedReason = computed(() => {
  if (!successorEinstufung.value && hasUnsavedCalculation.value) {
    return 'Es gibt ungespeicherte Änderungen. Bitte zuerst speichern – Brief und E-Mail zeigen den gespeicherten Stand.';
  }
  return null;
});
const previousEinstufungForPdf = computed(() =>
  successorEinstufung.value ? einstufung.value : sourceEinstufung.value ?? previousEinstufung.value
);

const parentFirstNames = computed(() => {
  return householdParents.value
    .map((parent) => parent.firstName?.trim())
    .filter((name): name is string => !!name)
    .slice(0, 2);
});

const greetingLine = computed(() => {
  if (parentFirstNames.value.length >= 2) {
    return `Hallo ${parentFirstNames.value[0]}, hallo ${parentFirstNames.value[1]},`;
  }
  if (parentFirstNames.value.length === 1) {
    return `Hallo ${parentFirstNames.value[0]},`;
  }
  return 'Hallo,';
});

const shortSummary = computed(() => {
  if (!einstufungForPdf.value) return '$zusammenfassungDerEinstufung';
  const e = einstufungForPdf.value;
  const childcareFeeText = e.monthlyChildcareFee > 0
    ? formatEuroAmount(e.monthlyChildcareFee, 'Euro')
    : `${formatEuroAmount(0, 'Euro')} (beitragsfrei)`;
  return `Kurz:
Platzgeld ${childcareFeeText}
Essensgeld ${formatEuroAmount(e.monthlyFoodFee, 'Euro')}`;
});

const membershipLine = computed(() => {
  if (!einstufungForPdf.value) return '$anweisungFürMitgliedsbeitrag';
  const fee = einstufungForPdf.value.annualMembershipFee;
  if (fee > 0) {
    return `Außerdem einmal pro Jahr die Mitgliedsgebühr für den Verein von ${
      formatEuroAmount(fee, 'Euro')
    }.`;
  }
  return 'Aktuell fällt keine jährliche Mitgliedsgebühr für den Verein an.';
});

const transferInstruction = computed(() => {
  return `Im Verwendungszweck sollte immer die Mitgliedsnummer ${childMemberNumber.value} angegeben werden.

${BANK_DETAILS}`;
});

const defaultEmailSubject = computed(() => {
  if (!einstufungForPdf.value) return 'Einstufung Elternbeiträge';
  const lastName = einstufungChild.value?.lastName || '';
  return `Einstufung Elternbeiträge ${einstufungForPdf.value.year}${lastName ? ` – ${lastName}` : ''}`;
});

const defaultEmailBody = computed(() => {
  const intro = einstufungForPdf.value?.sourceEinstufungId
    ? 'eure Einstufung haben wir angepasst.'
    : 'willkommen bei uns in der Kita!\n\nDanke für eure Unterlagen.';
  return `${greetingLine.value}

${intro}
Im Anhang findet ihr die Einstufung mit den monatlichen Beiträgen für die Kita als PDF.

${shortSummary.value}

${membershipLine.value}

${transferInstruction.value}
`;
});

function resetEmailDraft() {
  emailSubject.value = defaultEmailSubject.value;
  emailBody.value = defaultEmailBody.value;
  emailSubjectDirty.value = false;
  emailBodyDirty.value = false;
  mailError.value = null;
}

async function openLetter(mode: 'view' | 'download') {
  const target = einstufungForPdf.value;
  if (!target) return;
  mailError.value = null;
  // Open the tab synchronously so the popup blocker allows it.
  const tab = mode === 'view' ? window.open('', '_blank') : null;
  letterAction.value = mode;
  try {
    const file = await api.getEinstufungLetter(target.id);
    const url = URL.createObjectURL(file.blob);
    if (tab) {
      tab.location.href = url;
    } else {
      const link = document.createElement('a');
      link.href = url;
      link.download = file.filename ?? `Einstufung_${target.year}.pdf`;
      link.click();
    }
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (e) {
    tab?.close();
    mailError.value = e instanceof Error ? e.message : 'PDF konnte nicht erstellt werden';
  } finally {
    letterAction.value = null;
  }
}

async function sendLetter() {
  const target = einstufungForPdf.value;
  if (!target) return;
  mailError.value = null;
  mailSuccess.value = null;
  if (!emailRecipients.value.length) {
    mailError.value = 'Keine gültige E-Mail-Adresse bei den Eltern hinterlegt.';
    return;
  }
  const subject = emailSubject.value.trim();
  const body = emailBody.value.trim();
  if (!subject) {
    mailError.value = 'Bitte einen Betreff eintragen.';
    return;
  }
  if (!body) {
    mailError.value = 'Bitte einen E-Mail-Text eintragen.';
    return;
  }
  const again = target.letterSentAt ? ` Sie wurde bereits am ${formatDateTime(target.letterSentAt)} versendet.` : '';
  if (!confirm(`E-Mail mit der Einstufung als PDF an ${emailRecipients.value.join(', ')} senden?${again}`)) return;

  isSending.value = true;
  try {
    const result = await api.sendEinstufungLetter(target.id, { subject, body });
    target.letterSentAt = result.sentAt;
    mailSuccess.value = `E-Mail mit ${result.filename} am ${formatDateTime(result.sentAt)} an ${result.recipients.join(', ')} gesendet.`;
  } catch (e) {
    mailError.value = e instanceof Error ? e.message : 'E-Mail konnte nicht gesendet werden';
  } finally {
    isSending.value = false;
  }
}

async function loadData() {
  isLoading.value = true;
  error.value = null;
  einstufung.value = null;
  sourceEinstufung.value = null;
  previousEinstufung.value = null;
  successorEinstufung.value = null;
  expectationChanges.value = null;
  ++prefillSequence;
  existingOpenEinstufung.value = null;
  prefillSource.value = null;
  isPrefilling.value = false;
  try {
    const [childRes, householdRes] = await Promise.all([
      api.getChildren({ perPage: 2000 }),
      api.getHouseholds({ perPage: 2000 }),
    ]);
    children.value = childRes.data;
    households.value = householdRes.data;

    if (einstufungId.value) {
      const e = await api.getEinstufung(einstufungId.value);
      einstufung.value = e;
      if (e.sourceEinstufungId) {
        previousEinstufung.value = await api.getEinstufung(e.sourceEinstufungId);
      } else if (e.householdId) {
        const householdEinstufungen = await api.getEinstufungenForHousehold(e.householdId);
        successorEinstufung.value = householdEinstufungen.find((item) => item.sourceEinstufungId === e.id) ?? null;
      }
      selectedChildId.value = e.childId;
      selectedYear.value = e.year;
      // The form edits the actual change/admission date. `validFrom` is the
      // normalized billing-month boundary and must not replace that date.
      validFrom.value = (e.changeDate || e.validFrom).split('T')[0];
      childrenCount.value = e.childrenCount;
      incomeBasis.value = e.incomeBasis;
      provisional.value = e.provisional;
      notes.value = e.notes || '';
      parent1Income.value = { ...emptyIncome(), ...e.incomeCalculation.parent1 };
      parent2Income.value = { ...emptyIncome(), ...e.incomeCalculation.parent2 };
      savedFormSnapshot.value = formSnapshot.value;
    } else if (isFollowUpMode.value) {
      let source: Einstufung | null = null;
      if (route.query.sourceId) {
        source = await api.getEinstufung(route.query.sourceId as string);
      } else if (route.query.childId) {
        source = await api.getEinstufungForChild(route.query.childId as string);
      }
      if (!source) {
        throw new Error('Keine bestehende Einstufung als Vorlage gefunden.');
      }
      sourceEinstufung.value = source;
      selectedChildId.value = source.childId;
      selectedYear.value = new Date(calculateEffectiveFromMonth(todayISO())).getUTCFullYear();
      validFrom.value = todayISO();
      childrenCount.value = source.childrenCount;
      incomeBasis.value = source.incomeBasis;
      provisional.value = false;
      notes.value = source.notes || '';
      parent1Income.value = { ...emptyIncome(), ...source.incomeCalculation.parent1 };
      parent2Income.value = { ...emptyIncome(), ...source.incomeCalculation.parent2 };
    } else if (route.query.childId) {
      // Pre-select child from query param
      const childId = route.query.childId as string;
      selectedChildId.value = childId;
      // Ensure the child is in the list (fetch individually if needed)
      if (!children.value.find(c => c.id === childId)) {
        try {
          const child = await api.getChild(childId);
          children.value.unshift(child);
        } catch {
          // Child not found, ignore
        }
      }
      void applyChildDefaults();
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

async function handleSubmit() {
  if (!selectedChildId.value) {
    error.value = 'Bitte ein Kind auswählen';
    return;
  }

  isSaving.value = true;
  error.value = null;
  saveSuccess.value = false;

  const form = {
    incomeCalculation: incomeCalculation(),
    incomeBasis: incomeBasis.value,
    provisional: provisional.value,
    childrenCount: childrenCount.value,
    notes: notes.value || undefined,
  };

  try {
    if (!(await confirmPreviewWarnings())) return;
    if (isNew.value) {
      let created: Einstufung;
      if (isFollowUpMode.value) {
        if (!sourceEinstufung.value) {
          throw new Error('Keine bestehende Einstufung als Vorlage gefunden.');
        }
        const result = await api.createFollowUpEinstufung(sourceEinstufung.value.id, {
          ...form,
          changeDate: validFrom.value,
        });
        created = result.einstufung;
        expectationChanges.value = result.expectationChanges;
      } else {
        created = await api.createEinstufung({
          ...form,
          childId: selectedChildId.value,
          year: selectedYear.value,
        });
      }
      // Stay on page: show result + PDF immediately
      einstufung.value = created;
      router.replace(`/einstufungen/${created.id}`);
      saveSuccess.value = true;
      setTimeout(() => (saveSuccess.value = false), 3000);
    } else {
      const updated = await api.updateEinstufung(einstufungId.value!, form);
      einstufung.value = updated;
      savedFormSnapshot.value = formSnapshot.value;
      saveSuccess.value = true;
      setTimeout(() => (saveSuccess.value = false), 3000);
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isSaving.value = false;
  }
}

watch(
  () => route.fullPath,
  () => loadData(),
  { immediate: true },
);

watch(defaultEmailSubject, (next) => {
  if (!emailSubjectDirty.value) {
    emailSubject.value = next;
  }
}, { immediate: true });

watch(defaultEmailBody, (next) => {
  if (!emailBodyDirty.value) {
    emailBody.value = next;
  }
}, { immediate: true });
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center gap-4 mb-6">
      <button
        @click="router.push('/einstufungen')"
        class="p-2 rounded-lg hover:bg-accent transition-colors"
      >
        <ArrowLeft class="h-5 w-5" />
      </button>
      <div class="flex-1">
        <h1 class="text-2xl font-bold text-foreground">
          {{ isFollowUpMode ? 'Folgeeinstufung erstellen' : isNew ? 'Neue Einstufung' : 'Einstufung bearbeiten' }}
        </h1>
        <p v-if="einstufung && einstufung.child" class="text-sm text-muted-foreground mt-0.5">
          {{ einstufung.child.firstName }} {{ einstufung.child.lastName }} &middot; {{ einstufung.year }}
        </p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 text-primary animate-spin" />
    </div>

    <!-- Error -->
    <div v-if="error" class="bg-red-50 dark:bg-red-950/40 text-red-700 dark:text-red-300 p-4 rounded-lg mb-4">
      {{ error }}
    </div>

    <!-- Success -->
    <div v-if="saveSuccess" class="bg-green-50 dark:bg-green-950/40 text-green-700 dark:text-green-300 p-4 rounded-lg mb-4 flex items-center gap-2">
      <CheckCircle class="h-5 w-5" />
      {{ sourceEinstufung ? 'Folgeeinstufung erfolgreich gespeichert.' : 'Einstufung erfolgreich gespeichert.' }}
    </div>

    <form v-if="!isLoading" @submit.prevent="handleSubmit" class="space-y-6">
      <!-- Child & Parameters Card -->
      <div class="bg-card rounded-lg border shadow-sm p-6">
        <h2 class="text-lg font-semibold text-foreground mb-4 flex items-center gap-2">
          <User class="h-5 w-5 text-muted-foreground" />
          Kind & Parameter
        </h2>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <!-- Child search -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Kind *</label>
            <ChildSearchSelect
              v-model="selectedChildId"
              :children="children"
              :households="households"
              :disabled="!isNew || isFollowUpMode"
              @update:model-value="onChildSelected"
            />
          </div>

          <!-- Year -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Jahr *</label>
            <select
              v-model="selectedYear"
              :disabled="!isNew"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:bg-muted"
              required
            >
              <option v-for="y in yearOptions" :key="y" :value="y">
                {{ y }}
              </option>
            </select>
          </div>

          <!-- Income changes still need a date; initial validity comes from the child. -->
          <div v-if="isFollowUpMode">
            <label class="block text-sm font-medium text-foreground mb-1">Änderungsdatum *</label>
            <input
              type="date"
              v-model="validFrom"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
              required
            />
            <p v-if="isFollowUpMode" class="mt-1 text-xs text-muted-foreground">
              Wirksam ab {{ formatEffectiveFromMonth(effectiveFromMonthPreview) }}
            </p>
          </div>

          <div v-else class="md:col-span-2 rounded-lg border border-blue-100 bg-blue-50 dark:bg-blue-950/40 px-4 py-3 text-sm text-blue-800 dark:text-blue-300">
            Gültigkeit und Betreuungsstunden werden automatisch aus dem Kinddatensatz und dessen Stundenverlauf übernommen.
          </div>

          <!-- Children count -->
          <div>
            <label for="childrenCount" class="block text-sm font-medium text-foreground mb-1">Anzahl unterhaltspflichtiger Kinder *</label>
            <input
              id="childrenCount"
              type="number"
              v-model.number="childrenCount"
              min="1"
              max="10"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
              required
            />
            <p v-if="isNew && !isFollowUpMode && householdChildren.length" class="mt-1 text-xs text-muted-foreground">
              Im Haushalt bekannt: {{ householdChildren.map((c) => c.firstName).join(', ') }}<template
                v-if="selectedHousehold?.childrenCountForFees"> · zuletzt erfasst: {{ selectedHousehold.childrenCountForFees }}</template>.
              Weitere unterhaltspflichtige Kinder (z. B. Schulkinder) bitte hinzuzählen.
            </p>
          </div>

          <fieldset class="md:col-span-2 lg:col-span-3">
            <legend class="block text-sm font-medium text-foreground mb-2">Einkommensgrundlage *</legend>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
              <label
                v-for="option in incomeBasisOptions"
                :key="option.value"
                :class="[
                  'flex items-start gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
                  option.disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer hover:bg-accent',
                  incomeBasis === option.value ? 'border-primary bg-primary/5' : '',
                ]"
              >
                <input
                  type="radio"
                  name="incomeBasis"
                  :value="option.value"
                  v-model="incomeBasis"
                  :disabled="option.disabled"
                  class="mt-0.5 h-4 w-4 text-primary focus:ring-primary/20"
                />
                <span>
                  <span class="font-medium text-foreground">{{ option.label }}</span>
                  <span class="block text-xs text-muted-foreground">{{ option.hint }}</span>
                </span>
              </label>
            </div>
            <div class="mt-3 flex items-center gap-2">
              <input
                type="checkbox"
                v-model="provisional"
                id="provisional"
                class="h-4 w-4 text-primary rounded focus:ring-primary/20"
              />
              <label for="provisional" class="text-sm text-foreground">
                Vorläufig – Nachweise fehlen noch
              </label>
            </div>
          </fieldset>
        </div>

        <!-- Household info -->
        <div v-if="selectedHousehold" class="mt-4 p-3 bg-muted rounded-lg flex items-center gap-2 text-sm text-muted-foreground">
          <Home class="h-4 w-4 text-muted-foreground" />
          Haushalt: <strong>{{ selectedHousehold.name }}</strong>
          <span v-if="selectedHousehold.annualHouseholdIncome" class="ml-2">
            (bisheriges Einkommen: {{ formatCurrency(selectedHousehold.annualHouseholdIncome) }})
          </span>
        </div>
        <div v-else-if="childMissingHousehold" class="mt-4 p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-200 rounded-lg flex items-start gap-2 text-sm text-amber-800 dark:text-amber-300">
          <Info class="h-4 w-4 text-amber-500 dark:text-amber-400 mt-0.5 shrink-0" />
          <div>
            <strong>Kein Haushalt zugeordnet.</strong>
            Bitte zuerst Eltern anlegen und dem Kind einen Haushalt zuweisen, bevor eine Einstufung erstellt werden kann.
            <router-link
              :to="`/kinder/${selectedChildId}`"
              class="underline font-medium hover:text-amber-900 dark:text-amber-300"
            >
              Zum Kind →
            </router-link>
          </div>
        </div>

        <template v-if="isNew && !isFollowUpMode && selectedChild">
          <div v-if="isPrefilling" class="mt-4 flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin" />
            Bestehende Einstufungen des Haushalts werden geladen …
          </div>

          <div v-if="existingOpenEinstufung" class="mt-4 p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-200 rounded-lg flex items-start gap-2 text-sm text-amber-800 dark:text-amber-300">
            <Info class="h-4 w-4 text-amber-500 dark:text-amber-400 mt-0.5 shrink-0" />
            <div>
              <strong>Für {{ selectedChild.firstName }} gibt es bereits eine gültige Einstufung</strong>
              ({{ einstufungLabel(existingOpenEinstufung) }}). Bei geändertem Einkommen bitte eine
              Folgeeinstufung anlegen.
              <span class="mt-1 flex flex-wrap gap-3">
                <router-link :to="`/einstufungen/${existingOpenEinstufung.id}`" class="underline font-medium">
                  Einstufung öffnen →
                </router-link>
                <router-link :to="`/einstufungen/neu?sourceId=${existingOpenEinstufung.id}`" class="underline font-medium">
                  Folgeeinstufung erstellen →
                </router-link>
              </span>
            </div>
          </div>

          <div v-else-if="prefillSource" class="mt-4 p-3 bg-blue-50 dark:bg-blue-950/40 border border-blue-100 rounded-lg flex items-start gap-2 text-sm text-blue-800 dark:text-blue-300">
            <Info class="h-4 w-4 mt-0.5 shrink-0" />
            <div class="flex-1">
              {{ prefillSummary }} aus der Einstufung von {{ einstufungLabel(prefillSource) }}.
              Bitte prüfen, ob die Angaben noch aktuell sind.
            </div>
            <button type="button" class="shrink-0 text-xs font-medium underline" @click="discardPrefill">
              Verwerfen
            </button>
          </div>

          <div v-if="isKindergartenAtStart" class="mt-4 p-3 bg-green-50 dark:bg-green-950/40 border border-green-200 rounded-lg flex items-start gap-2 text-sm text-green-800 dark:text-green-300">
            <Info class="h-4 w-4 mt-0.5 shrink-0" />
            <div>
              {{ selectedChild.firstName }} ist ab Beginn der Einstufung im Kindergartenalter (geb.
              {{ formatDate(selectedChild.birthDate) }}). Die Betreuung ist beitragsfrei, das Einkommen
              hat keinen Einfluss auf den Beitrag.
            </div>
          </div>
        </template>

        <!-- Notes -->
        <div class="mt-4">
          <label class="block text-sm font-medium text-foreground mb-1">Bemerkungen</label>
          <textarea
            v-model="notes"
            rows="2"
            class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
            placeholder="Optionale Bemerkungen..."
          ></textarea>
        </div>
      </div>

      <!-- Income Calculation Card -->
      <div v-if="incomeBasis === 'INCOME'" class="bg-card rounded-lg border shadow-sm p-6">
        <h2 class="text-lg font-semibold text-foreground mb-1 flex items-center gap-2">
          <Calculator class="h-5 w-5 text-muted-foreground" />
          Einkommensberechnung
        </h2>
        <p class="text-sm text-muted-foreground mb-4">
          Festsetzung des Elternbeitrages – alle Beträge als Jahressummen in EUR
        </p>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <!-- Parent 1 -->
          <section aria-label="Elternteil 1">
            <h3 class="font-medium text-foreground mb-3 pb-2 border-b">
              Elternteil 1 (Mutter / Sorgeberechtigte/r)
            </h3>
            <IncomeForm v-model="parent1Income" />
          </section>

          <!-- Parent 2 -->
          <section aria-label="Elternteil 2">
            <h3 class="font-medium text-foreground mb-3 pb-2 border-b">
              Elternteil 2 (Vater / Sorgeberechtigte/r)
            </h3>
            <IncomeForm v-model="parent2Income" />
          </section>
        </div>

        <!-- Income from the live preview -->
        <div v-if="preview" class="mt-6 p-4 bg-blue-50 dark:bg-blue-950/40 rounded-lg border border-blue-100">
          <h4 class="text-sm font-semibold text-blue-900 dark:text-blue-300 mb-3 flex items-center gap-2">
            <Euro class="h-4 w-4" />
            Einkommensvorschau
          </h4>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
            <div>
              <span class="text-blue-600 dark:text-blue-300">Elternteil 1 (Netto)</span>
              <p class="font-semibold text-blue-900 dark:text-blue-300">{{ formatCurrency(preview.income.parent1NetIncome) }}</p>
              <span class="block text-xs text-blue-500 dark:text-blue-400">Beitragsrelevant: {{ formatCurrency(preview.income.parent1FeeRelevantIncome) }}</span>
              <span v-if="preview.income.parent1AdvertisingCosts > 0" class="block text-xs text-blue-500 dark:text-blue-400">
                WK-Pauschale angerechnet: {{ formatCurrency(preview.income.parent1AdvertisingCosts) }}
              </span>
            </div>
            <div>
              <span class="text-blue-600 dark:text-blue-300">Elternteil 2 (Netto)</span>
              <p class="font-semibold text-blue-900 dark:text-blue-300">{{ formatCurrency(preview.income.parent2NetIncome) }}</p>
              <span class="block text-xs text-blue-500 dark:text-blue-400">Beitragsrelevant: {{ formatCurrency(preview.income.parent2FeeRelevantIncome) }}</span>
              <span v-if="preview.income.parent2AdvertisingCosts > 0" class="block text-xs text-blue-500 dark:text-blue-400">
                WK-Pauschale angerechnet: {{ formatCurrency(preview.income.parent2AdvertisingCosts) }}
              </span>
            </div>
            <div>
              <span class="text-blue-600 dark:text-blue-300">Haushaltseinkommen (beitragsrelevant)</span>
              <p class="text-lg font-bold text-blue-900 dark:text-blue-300">{{ formatCurrency(preview.income.householdFeeIncome) }}</p>
              <span class="text-xs text-blue-500 dark:text-blue-400">Gesamt inkl. Leistungen: {{ formatCurrency(preview.income.householdFullIncome) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Highest rate notice -->
      <div v-if="incomeBasis === 'HIGHEST_RATE'" class="bg-orange-50 dark:bg-orange-950/40 border border-orange-200 rounded-lg p-4 flex items-start gap-3">
        <Info class="h-5 w-5 text-orange-500 dark:text-orange-400 mt-0.5 shrink-0" />
        <div class="text-sm text-orange-800 dark:text-orange-300">
          <strong>Freiwillige Anerkennung des Höchstsatzes</strong>
          <p class="mt-1">
            Die Einkommensberechnung entfällt, da der Höchstsatz freiwillig anerkannt wird.
            Der Beitrag wird direkt nach dem Höchstsatz der Satzungstabelle berechnet.
          </p>
        </div>
      </div>

      <div v-if="incomeBasis === 'NOT_REQUIRED'" class="bg-green-50 dark:bg-green-950/40 border border-green-200 rounded-lg p-4 flex items-start gap-3">
        <Info class="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5 shrink-0" />
        <div class="text-sm text-green-800 dark:text-green-300">
          <strong>Einkommen nicht erforderlich</strong>
          <p class="mt-1">
            Das Kind ist im Kindergartenalter, die Betreuung ist beitragsfrei. Ein Einkommen wird nicht
            erfasst; in Einstufung und Haushalt bleibt es leer.
          </p>
        </div>
      </div>

      <!-- Result: the stored Einstufung, or the live preview while the form differs from it -->
      <div v-if="shownResult || isPreviewing || previewError || preview?.warnings.length" class="bg-card rounded-lg border shadow-sm p-6">
        <div class="flex items-center justify-between gap-3 mb-4">
          <h2 class="text-lg font-semibold text-foreground flex items-center gap-2">
            <FileText class="h-5 w-5 text-muted-foreground" />
            {{ hasUnsavedCalculation ? 'Vorschau – noch nicht gespeichert' : 'Ergebnis' }}
            <Loader2 v-if="isPreviewing" class="h-4 w-4 animate-spin text-muted-foreground" />
          </h2>
          <button
            v-if="einstufung && !isNew && !successorEinstufung && !hasUnsavedCalculation"
            type="button"
            @click="router.push(`/einstufungen/neu?sourceId=${einstufung.id}`)"
            class="inline-flex items-center gap-2 px-3 py-2 text-xs text-primary-foreground bg-primary rounded-lg hover:bg-primary/90 transition-colors"
          >
            Folgeeinstufung erstellen
          </button>
        </div>

        <p v-if="previewError" class="mb-4 text-sm text-destructive">{{ previewError }}</p>

        <div v-if="hasUnsavedCalculation && preview && (preview.warnings.length || preview.notices.length)" class="mb-4 space-y-2">
          <div
            v-for="warning in preview.warnings"
            :key="warning"
            class="p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-200 rounded-lg flex items-start gap-2 text-sm text-amber-800 dark:text-amber-300"
          >
            <AlertTriangle class="h-4 w-4 mt-0.5 shrink-0" />
            {{ warning }}
          </div>
          <div
            v-for="notice in preview.notices"
            :key="notice"
            class="p-3 bg-blue-50 dark:bg-blue-950/40 border border-blue-100 rounded-lg flex items-start gap-2 text-sm text-blue-800 dark:text-blue-300"
          >
            <Info class="h-4 w-4 mt-0.5 shrink-0" />
            {{ notice }}
          </div>
        </div>

        <template v-if="shownResult">
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <span class="text-sm text-muted-foreground">Platzgeld</span>
              <p class="text-xl font-bold text-foreground">{{ formatCurrency(shownResult.monthlyChildcareFee) }}</p>
              <span class="text-xs text-muted-foreground">monatlich</span>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Essengeld</span>
              <p class="text-xl font-bold text-foreground">{{ formatCurrency(shownResult.monthlyFoodFee) }}</p>
              <span class="text-xs text-muted-foreground">monatlich</span>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Vereinsbeitrag</span>
              <p class="text-xl font-bold text-foreground">{{ formatCurrency(shownResult.annualMembershipFee) }}</p>
              <span class="text-xs text-muted-foreground">jährlich</span>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Regel</span>
              <p class="text-lg font-semibold text-foreground">{{ shownResult.feeRule }}</p>
              <span v-if="shownResult.discountPercent > 0" class="text-xs text-green-600 dark:text-green-300">
                {{ shownResult.discountPercent }}% Geschwisterrabatt
              </span>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap gap-2 text-xs">
            <span class="px-2 py-1 rounded-full bg-muted text-muted-foreground">
              {{ INCOME_BASIS_LABELS[shownResult.incomeBasis] }}<template
                v-if="shownResult.incomeBasis === 'INCOME' && shownResult.annualNetIncome != null">:
                {{ formatCurrency(shownResult.annualNetIncome) }} / Jahr</template>
            </span>
            <span v-if="shownResult.provisional" class="px-2 py-1 rounded-full bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300">
              Vorläufig – Nachweise fehlen
            </span>
          </div>

          <!-- Monthly table -->
          <div v-if="shownResult.monthlyTable && shownResult.monthlyTable.length" class="mt-6">
            <h3 class="text-sm font-medium text-foreground mb-2">Monatsübersicht</h3>
            <table class="min-w-full divide-y divide-border text-sm">
              <thead class="bg-muted">
                <tr>
                  <th class="px-3 py-2 text-left text-xs font-medium text-muted-foreground uppercase">Monat</th>
                  <th class="px-3 py-2 text-left text-xs font-medium text-muted-foreground uppercase">Bereich</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-muted-foreground uppercase">Platzgeld</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-muted-foreground uppercase">Essengeld</th>
                  <th class="px-3 py-2 text-right text-xs font-medium text-muted-foreground uppercase">Vereinsbeitrag</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-for="row in shownResult.monthlyTable" :key="`${row.year}-${row.month}`">
                  <td class="px-3 py-2 text-foreground">
                    {{ formatMonthYear(row.year, row.month) }}
                  </td>
                  <td class="px-3 py-2 text-muted-foreground">{{ row.careType }} · {{ row.careHoursPerWeek }}h</td>
                  <td class="px-3 py-2 text-right font-medium">{{ formatCurrency(row.childcareFee) }}</td>
                  <td class="px-3 py-2 text-right">{{ formatCurrency(row.foodFee) }}</td>
                  <td class="px-3 py-2 text-right">{{ row.membershipFee > 0 ? formatCurrency(row.membershipFee) : '—' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </div>

      <div v-if="expectationChanges" class="bg-card rounded-lg border shadow-sm p-6">
        <h2 class="text-lg font-semibold text-foreground mb-4 flex items-center gap-2">
          <CheckCircle class="h-5 w-5 text-muted-foreground" />
          Beitragsanpassung
        </h2>
        <div class="grid grid-cols-2 md:grid-cols-5 gap-4 text-sm">
          <div>
            <span class="text-muted-foreground">Neu angelegt</span>
            <p class="text-xl font-bold text-foreground">{{ expectationChanges.created }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">Aktualisiert</span>
            <p class="text-xl font-bold text-foreground">{{ expectationChanges.updated }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">Übersprungen</span>
            <p class="text-xl font-bold text-foreground">{{ expectationChanges.skipped }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">Offene Differenz</span>
            <p class="text-xl font-bold text-foreground">{{ formatCurrency(expectationChanges.deltaOpen) }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">Gutschrift prüfen</span>
            <p class="text-xl font-bold text-foreground">{{ expectationChanges.creditReviewRequired.length }}</p>
          </div>
        </div>

        <div v-if="expectationChanges.creditReviewRequired.length" class="mt-5 overflow-x-auto">
          <table class="min-w-full divide-y divide-border text-sm">
            <thead class="bg-amber-50 dark:bg-amber-950/40">
              <tr>
                <th class="px-3 py-2 text-left text-xs font-medium text-amber-800 dark:text-amber-300 uppercase">Monat</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-amber-800 dark:text-amber-300 uppercase">Alt</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-amber-800 dark:text-amber-300 uppercase">Neu</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-amber-800 dark:text-amber-300 uppercase">Bezahlt</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-amber-800 dark:text-amber-300 uppercase">Prüfbetrag</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <tr v-for="item in expectationChanges.creditReviewRequired" :key="item.feeId">
                <td class="px-3 py-2 text-foreground">{{ formatMonthYear(item.year, item.month) }}</td>
                <td class="px-3 py-2 text-right">{{ formatCurrency(item.oldAmount) }}</td>
                <td class="px-3 py-2 text-right">{{ formatCurrency(item.newAmount) }}</td>
                <td class="px-3 py-2 text-right">{{ formatCurrency(item.matchedAmount) }}</td>
                <td class="px-3 py-2 text-right font-medium text-amber-800 dark:text-amber-300">{{ formatCurrency(item.creditAmount) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-if="einstufungForPdf" class="bg-card rounded-lg border shadow-sm p-6 space-y-4">
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-lg font-semibold text-foreground flex items-center gap-2">
            <Mail class="h-5 w-5 text-muted-foreground" />
            Brief und E-Mail an Eltern
          </h2>
          <button
            type="button"
            @click="resetEmailDraft"
            class="inline-flex items-center gap-2 px-3 py-2 text-xs text-foreground bg-muted rounded-lg hover:bg-accent transition-colors"
          >
            <RotateCcw class="h-3.5 w-3.5" />
            Standardtext einsetzen
          </button>
        </div>

        <p v-if="successorEinstufung" class="text-sm text-muted-foreground">
          Brief und E-Mail beziehen sich auf die neueste Einstufung ({{ einstufungLabel(successorEinstufung) }}), mit dieser
          Einstufung als bisherigem Beitrag.
        </p>
        <div v-if="letterBlockedReason" class="text-sm text-amber-800 dark:text-amber-300 bg-amber-50 dark:bg-amber-950/40 border border-amber-200 rounded-lg px-3 py-2">
          {{ letterBlockedReason }}
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            :disabled="!!letterBlockedReason || letterAction !== null"
            @click="openLetter('view')"
            class="inline-flex items-center gap-2 px-3 py-2 text-sm text-foreground bg-muted rounded-lg hover:bg-accent transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="letterAction === 'view'" class="h-4 w-4 animate-spin" />
            <ExternalLink v-else class="h-4 w-4" />
            PDF ansehen
          </button>
          <button
            type="button"
            :disabled="!!letterBlockedReason || letterAction !== null"
            @click="openLetter('download')"
            class="inline-flex items-center gap-2 px-3 py-2 text-sm text-foreground bg-muted rounded-lg hover:bg-accent transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="letterAction === 'download'" class="h-4 w-4 animate-spin" />
            <Download v-else class="h-4 w-4" />
            PDF herunterladen
          </button>
          <EinstufungPDF
            :einstufung="einstufungForPdf"
            :previous-einstufung="previousEinstufungForPdf"
          />
          <span class="text-xs text-muted-foreground">
            {{ einstufungForPdf.letterSentAt
              ? `Zuletzt per E-Mail versendet am ${formatDateTime(einstufungForPdf.letterSentAt)}.`
              : 'Noch nicht per E-Mail versendet.' }}
          </span>
        </div>

        <div>
          <label class="block text-sm font-medium text-foreground mb-1">Empfänger</label>
          <p v-if="emailRecipients.length" class="text-sm text-foreground">
            {{ emailRecipients.join(', ') }}
          </p>
          <p v-else class="text-sm text-amber-700 dark:text-amber-300">
            Keine Empfänger gefunden. Bitte E-Mail-Adressen in den Elterndaten hinterlegen.
          </p>
        </div>

        <div>
          <label for="email-subject" class="block text-sm font-medium text-foreground mb-1">Betreff</label>
          <input
            id="email-subject"
            v-model="emailSubject"
            @input="emailSubjectDirty = true"
            type="text"
            class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
          />
        </div>

        <div>
          <label for="email-body" class="block text-sm font-medium text-foreground mb-1">Text</label>
          <textarea
            id="email-body"
            v-model="emailBody"
            @input="emailBodyDirty = true"
            rows="16"
            class="w-full border rounded-lg px-3 py-2 text-sm leading-6 focus:ring-2 focus:ring-primary/20 focus:border-primary"
          ></textarea>
          <p class="mt-2 text-xs text-muted-foreground">
            Der Brief wird als PDF angehängt. Der Versand wird im E-Mail-Protokoll festgehalten.
          </p>
        </div>

        <div v-if="mailError" class="text-sm text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-950/40 border border-red-100 rounded-lg px-3 py-2">
          {{ mailError }}
        </div>
        <div v-if="mailSuccess" class="text-sm text-green-800 dark:text-green-300 bg-green-50 dark:bg-green-950/40 border border-green-200 rounded-lg px-3 py-2">
          {{ mailSuccess }}
        </div>

        <div>
          <button
            type="button"
            :disabled="!!letterBlockedReason || isSending || !emailRecipients.length"
            @click="sendLetter"
            class="inline-flex items-center gap-2 px-4 py-2 text-sm text-primary-foreground bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isSending" class="h-4 w-4 animate-spin" />
            <Send v-else class="h-4 w-4" />
            E-Mail mit PDF senden
          </button>
        </div>
      </div>

      <!-- Submit -->
      <div class="flex items-center justify-end">
        <div class="flex items-center gap-3">
        <button
          type="button"
          @click="router.push('/einstufungen')"
          class="px-4 py-2 text-sm text-foreground bg-muted rounded-lg hover:bg-accent transition-colors"
        >
          Abbrechen
        </button>
        <button
          type="submit"
          :disabled="isSaving || isPrefilling || childMissingHousehold || (isNew && !!existingOpenEinstufung)"
          class="inline-flex items-center gap-2 px-6 py-2 text-sm text-primary-foreground bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          <Save v-else class="h-4 w-4" />
          {{ isFollowUpMode ? 'Folgeeinstufung speichern' : isNew ? 'Einstufung speichern' : 'Änderungen speichern' }}
        </button>
        </div>
      </div>
    </form>
  </div>
</template>
