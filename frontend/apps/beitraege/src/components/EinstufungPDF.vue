<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import type { Einstufung, Child } from '@/api/types';
import { FileDown, Loader2 } from 'lucide-vue-next';
import printStyles from './EinstufungPDF.css?raw';
import logoUrl from '@/assets/knirpsenstadt-logo.png';

const props = defineProps<{
  einstufung: Einstufung;
  previousEinstufung?: Einstufung | null;
}>();

const isGenerating = ref(false);
const pdfContainer = ref<HTMLElement | null>(null);
const logoSrc = ref<string>(logoUrl);

const child = computed(() => props.einstufung.child as Child | undefined);

// Compute the distinct fee columns for the letter
interface FeeColumn {
  label: string; // e.g. "Sept 25"
  careHours: number;
  careType: string; // "Krippe" or "Kindergarten"
  childcareFee: number;
  foodFee: number;
  membershipFee: number;
}

function parseMonthStart(value?: string): Date | null {
  if (!value) return null;
  const [year, month] = value.split('T')[0].split('-').map(Number);
  if (!year || !month) return null;
  return new Date(Date.UTC(year, month - 1, 1));
}

function addUtcMonths(date: Date, months: number): Date {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + months, 1));
}

function isSameOrAfterMonth(left: Date, right: Date): boolean {
  if (left.getUTCFullYear() !== right.getUTCFullYear()) {
    return left.getUTCFullYear() > right.getUTCFullYear();
  }
  return left.getUTCMonth() >= right.getUTCMonth();
}

function formatCareType(ct: string) {
  if (ct === 'krippe') return 'Krippe';
  if (ct === 'kindergarten') return 'Kindergarten';
  return ct.charAt(0).toUpperCase() + ct.slice(1);
}

function formatMonth(month: number, year: number) {
  return new Date(Date.UTC(year, month, 1)).toLocaleString('de-DE', {
    month: 'short',
    year: '2-digit',
    timeZone: 'UTC',
  });
}

const feeColumns = computed<FeeColumn[]>(() => {
  const e = props.einstufung;
  const validFrom = parseMonthStart(e.effectiveFromMonth || e.validFrom) ?? new Date();
  const startMonth = validFrom.getUTCMonth(); // 0-based
  const startYear = validFrom.getUTCFullYear();

  const cols: FeeColumn[] = [];

  if (e.sourceEinstufungId && props.previousEinstufung) {
    const previous = props.previousEinstufung;
    const previousMonth = addUtcMonths(validFrom, -1);
    const previousStart = parseMonthStart(previous.effectiveFromMonth || previous.validFrom);

    if (previousStart && isSameOrAfterMonth(previousMonth, previousStart)) {
      const previousRow = previous.monthlyTable?.find((row) =>
        row.year === previousMonth.getUTCFullYear() && row.month === previousMonth.getUTCMonth() + 1
      );

      cols.push({
        label: formatMonth(previousMonth.getUTCMonth(), previousMonth.getUTCFullYear()),
        careHours: previousRow?.careHoursPerWeek ?? previous.careHoursPerWeek,
        careType: previousRow?.careType ?? formatCareType(previous.careType),
        childcareFee: previousRow?.childcareFee ?? previous.monthlyChildcareFee,
        foodFee: previousRow?.foodFee ?? previous.monthlyFoodFee,
        membershipFee: 0,
      });
    }
  }

  // Column 1: First month (with membership fee)
  cols.push({
    label: formatMonth(startMonth, startYear),
    careHours: e.careHoursPerWeek,
    careType: formatCareType(e.careType),
    childcareFee: e.monthlyChildcareFee,
    foodFee: e.monthlyFoodFee,
    membershipFee: e.annualMembershipFee,
  });

  // Column 2: Second month (no membership fee)
  const m2 = startMonth + 1;
  const y2 = m2 > 11 ? startYear + 1 : startYear;
  cols.push({
    label: formatMonth(m2 % 12, y2),
    careHours: e.careHoursPerWeek,
    careType: formatCareType(e.careType),
    childcareFee: e.monthlyChildcareFee,
    foodFee: e.monthlyFoodFee,
    membershipFee: 0,
  });

  // Column 3: If child turns 3 within the next 12 months → beitragsfrei
  if (child.value && e.careType === 'krippe') {
    const birthDate = new Date(child.value.birthDate);
    const turnsThreeDate = new Date(birthDate.getFullYear() + 3, birthDate.getMonth(), birthDate.getDate());
    // The month the child transitions to Kindergarten (first full month after turning 3)
    let transMonth = turnsThreeDate.getMonth();
    let transYear = turnsThreeDate.getFullYear();
    // If birthday is not the first of the month, transition happens next month
    if (turnsThreeDate.getDate() > 1) {
      transMonth += 1;
      if (transMonth > 11) {
        transMonth = 0;
        transYear += 1;
      }
    }

    const transDate = new Date(transYear, transMonth, 1);
    const windowEnd = new Date(startYear, startMonth + 12, 1);

    if (transDate > validFrom && transDate <= windowEnd) {
      cols.push({
        label: formatMonth(transMonth, transYear),
        careHours: e.careHoursPerWeek,
        careType: 'Kindergarten',
        childcareFee: 0,
        foodFee: e.monthlyFoodFee,
        membershipFee: 0,
      });
    }
  }

  return cols;
});

const isFollowUp = computed(() => !!props.einstufung.sourceEinstufungId);
const showMembershipRow = computed(() => isFollowUp.value || feeColumns.value.some(c => c.membershipFee > 0));
const primaryFeeColumnIndex = computed(() => feeColumns.value.length > 2 && isFollowUp.value ? 1 : 0);
const isPrimaryFeeColumn = (idx: number) => idx === primaryFeeColumnIndex.value;
const monthlyTotal = (col: FeeColumn) => col.childcareFee + col.foodFee;

const entryDateFormatted = computed(() => {
  if (!child.value?.entryDate) return '—';
  return new Date(child.value.entryDate).toLocaleDateString('de-DE');
});

const birthDateFormatted = computed(() => {
  if (!child.value?.birthDate) return '—';
  return new Date(child.value.birthDate).toLocaleDateString('de-DE');
});

const documentDateFormatted = computed(() => {
  const raw = props.einstufung.createdAt;
  const date = raw ? new Date(raw) : new Date();
  const value = Number.isNaN(date.getTime()) ? new Date() : date;
  return value.toLocaleDateString('de-DE', { day: '2-digit', month: 'long', year: 'numeric' });
});

const validFromFormatted = computed(() => {
  const start = parseMonthStart(props.einstufung.effectiveFromMonth || props.einstufung.validFrom);
  if (!start) return '—';
  return start.toLocaleDateString('de-DE', { month: 'long', year: 'numeric', timeZone: 'UTC' });
});

const memberNumber = computed(() => child.value?.memberNumber ?? '—');

const childName = computed(() => {
  if (!child.value) return 'Vorname Nachname';
  return `${child.value.firstName} ${child.value.lastName}`;
});

const einstufungYear = computed(() => props.einstufung.year);

const feeRuleText = computed(() => {
  const e = props.einstufung;
  if (e.highestRateVoluntary) {
    return 'Die Einstufung erfolgte aufgrund der freiwilligen Anerkennung des Höchstsatzes.';
  }
  if (e.feeRule === 'beitragsfrei') {
    return `Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise vorgenommen. Gemäß Elternentlastungsgesetz ist der Beitrag beitragsfrei. Bei Änderungen informiert uns bitte umgehend.`;
  }
  const ruleRef = e.feeRule.includes('Entlastung')
    ? 'nach dem Elternentlastungsgesetz 2023/2024'
    : 'nach der Elternbeitragssatzung';
  return `Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise und ${ruleRef} vorgenommen. Bei Änderungen informiert uns bitte umgehend.`;
});

function formatEur(amount: number): string {
  return amount.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €';
}

let inlineLogoPromise: Promise<string> | null = null;

// The print window is a blank document, so relative asset URLs would not
// resolve there. Inline the logo as a data URL (with an absolute URL fallback).
function loadInlineLogo(): Promise<string> {
  if (!inlineLogoPromise) {
    const absoluteUrl = new URL(logoUrl, window.location.href).href;
    inlineLogoPromise = fetch(absoluteUrl)
      .then((response) => (response.ok ? response.blob() : Promise.reject(new Error('logo request failed'))))
      .then(
        (blob) =>
          new Promise<string>((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve(String(reader.result));
            reader.onerror = () => reject(reader.error);
            reader.readAsDataURL(blob);
          })
      )
      .catch(() => absoluteUrl);
  }
  return inlineLogoPromise;
}

async function waitForPdfLayout() {
  await nextTick();

  if ('fonts' in document) {
    try {
      await document.fonts.ready;
    } catch {
      // Ignore font loading issues and continue with the current layout state.
    }
  }

  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => resolve());
    });
  });
}

async function generatePdf() {
  if (!pdfContainer.value) return;
  isGenerating.value = true;

  try {
    logoSrc.value = await loadInlineLogo();
    await waitForPdfLayout();
    const printWindow = window.open('', '_blank', 'width=960,height=1200');
    if (!printWindow) {
      console.error('Print window could not be opened');
      return;
    }

    const safeChildName = childName.value
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');

    const title = `Einstufung_${einstufungYear.value}_${safeChildName.replace(/\s+/g, '_')}`;
    const documentHtml = `<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${title}</title>
  <style>
${printStyles}

@page {
  size: A4;
  margin: 13mm 15mm 10mm 15mm;
}

html,
body {
  margin: 0;
  padding: 0;
  background: #fff;
}

body {
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
  padding: 0;
}

.print-root {
  padding: 0;
  margin: 0;
}

  </style>
</head>
<body>
  <div class="print-root">${pdfContainer.value.outerHTML}</div>
  <script>
    window.addEventListener('load', function () {
      window.focus();
      setTimeout(function () {
        window.print();
      }, 40);
    });
    window.addEventListener('afterprint', function () {
      window.close();
    });
  <\/script>
</body>
</html>`;

    printWindow.document.open();
    printWindow.document.write(documentHtml);
    printWindow.document.close();
  } finally {
    isGenerating.value = false;
  }
}

defineExpose({ generatePdf });
</script>

<template>
  <div>
    <!-- Download button -->
    <button
      @click="generatePdf"
      :disabled="isGenerating"
      class="inline-flex items-center gap-2 px-4 py-2 text-sm text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
    >
      <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
      <FileDown v-else class="h-4 w-4" />
      PDF drucken / speichern
    </button>

    <!-- Hidden print layout -->
    <div class="pdf-stage" aria-hidden="true">
      <div ref="pdfContainer" class="page">

        <!-- Briefkopf -->
        <header>
          <div class="letterhead">
            <div class="letterhead__org">
              <div class="letterhead__name">Kita Knirpsenstadt e.&thinsp;V.</div>
              <div class="letterhead__claim">Elternverein &middot; Der Vorstand</div>
            </div>
            <div class="letterhead__mark">
              <img class="letterhead__logo" :src="logoSrc" alt="Kita Knirpsenstadt e. V." />
            </div>
          </div>
          <div class="letterhead__rule"></div>
        </header>

        <!-- Bezugszeile / Datum -->
        <div class="meta">
          <div class="meta__left">Mitgliedsnummer {{ memberNumber }}</div>
          <div class="meta__right">Panketal, den {{ documentDateFormatted }}</div>
        </div>

        <!-- Betreff -->
        <h1 class="subject">Einstufung der Elternbeiträge {{ einstufungYear }}</h1>
        <div class="subject__sub">
          für {{ childName }} &middot; gültig ab {{ validFromFormatted }}
        </div>

        <p class="salutation">Liebe Eltern,</p>
        <p class="body-text">
          auf Grundlage der eingereichten Nachweise und der Elternbeitragsordnung des Trägers haben
          wir die Elternbeiträge für euer Kind wie folgt eingestuft.
        </p>

        <!-- Angaben zum Kind -->
        <section class="section">
          <div class="section__heading">Angaben zum Kind</div>
          <table class="data-table">
            <tbody>
              <tr>
                <th>Name</th>
                <td>{{ childName }}</td>
                <th>Geburtsdatum</th>
                <td>{{ birthDateFormatted }}</td>
              </tr>
              <tr>
                <th>Besucht die Kita seit</th>
                <td>{{ entryDateFormatted }}</td>
                <th>Mitgliedsnummer</th>
                <td>{{ memberNumber }}</td>
              </tr>
              <tr>
                <th>Einrichtung</th>
                <td colspan="3">Kita Knirpsenstadt e.&thinsp;V., Ahornallee 27, 16341 Panketal</td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- Grundlage der Einstufung -->
        <div class="callout">
          <span class="callout__lead">Grundlage der Einstufung:</span>
          {{ feeRuleText }}
        </div>

        <!-- Beitragsübersicht -->
        <section class="section">
          <div class="section__heading">Monatliche Beiträge</div>

          <table class="fee-table">
            <thead>
              <tr>
                <th class="fee-table__col-label">&nbsp;</th>
                <th
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  :class="{ 'fee-table__col-month--primary': isPrimaryFeeColumn(idx) }"
                >
                  {{ col.label }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="fee-table__row-label">Betreuungsbereich</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': isPrimaryFeeColumn(idx) }"
                >
                  {{ col.careType }}
                </td>
              </tr>
              <tr>
                <td class="fee-table__row-label">Betreuungszeit je Woche</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': isPrimaryFeeColumn(idx) }"
                >
                  {{ col.careHours }} Std.
                </td>
              </tr>
              <tr>
                <td class="fee-table__row-label">Platzgeld (monatlich)</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': isPrimaryFeeColumn(idx) }"
                >
                  {{ formatEur(col.childcareFee) }}
                </td>
              </tr>
              <tr>
                <td class="fee-table__row-label">Essensgeld (monatlich)</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': isPrimaryFeeColumn(idx) }"
                >
                  {{ formatEur(col.foodFee) }}
                </td>
              </tr>
              <tr class="fee-table__row--total">
                <td class="fee-table__row-label">Monatlich zu zahlen</td>
                <td
                  v-for="col in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                >
                  {{ formatEur(monthlyTotal(col)) }}
                </td>
              </tr>
              <tr v-if="showMembershipRow" class="fee-table__row--membership">
                <td class="fee-table__row-label">
                  Vereinsbeitrag (jährlich)
                  <template v-if="isFollowUp"> &ndash; bereits bezahlt</template>
                </td>
                <td
                  v-for="col in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                >
                  {{ col.membershipFee > 0 || isFollowUp ? formatEur(col.membershipFee) : '—' }}
                </td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- Hinweis zur Zahlungsweise -->
        <div class="callout callout--quiet">
          <span class="callout__lead">Hinweis:</span>
          Bitte gleicht die Beträge für Mitgliedschaft, Betreuung und Essensgeld in
          <strong>getrennten</strong> Zahlungen unter Angabe des Namens des Kindes und der
          Mitgliedsnummer aus. Die Beträge gelten fortlaufend ab dem jeweils genannten Monat,
          bis eine neue Einstufung erfolgt.
        </div>

        <!-- Kleingedrucktes -->
        <div class="fineprint">
          <div class="fineprint__col">
            <div class="fineprint__heading">Rechtliche Grundlage</div>
            <p>
              Nach § 17 des Kindertagesstättengesetzes haben die Erziehungsberechtigten Beiträge zur
              Inanspruchnahme eines Platzes in der Kindertagesstätte zu entrichten. Dieser monatliche
              Elternbeitrag wird in Verbindung mit der Elternbeitragsordnung des Trägers ermittelt.
              Die Kindertagesstätte „Knirpsenstadt“ in 16341 Panketal, Ahornallee 27 befindet sich in
              freier Trägerschaft des „Knirpsenstadt e.&thinsp;V. Panketal“.
            </p>
            <p>
              Berechnet wird nach wirtschaftlicher Leistungsfähigkeit (Nettoeinkommen im Jahr), dem
              Alter des Kindes und der beanspruchten Betreuungszeit. Eine Ermäßigung des
              Elternbeitrages wird auch nach der Anzahl der unterhaltspflichtigen Kinder gewährt
              (jedoch nicht nach dem Brandenburger Entlastungspaket).
            </p>
          </div>
          <div class="fineprint__col">
            <div class="fineprint__heading">Zahlungsbedingungen</div>
            <p>
              Der monatliche Beitrag wird am 5. eines jeden Monats fällig. Beiträge, die in Verzug
              sind, werden zusätzlich mit einer Mahngebühr von 10,00 € erhoben. Der Vereinsbeitrag
              (derzeit 30,00 €) ist jährlich zu zahlen: bei Vertragsbeginn sofort, ansonsten bis
              spätestens Ende des ersten Quartals. Nach Fristablauf wird ein Mahngeld von 5,00 €
              erhoben.
            </p>
            <div class="fineprint__heading fineprint__heading--sub">Änderungspflicht</div>
            <p>
              Wenn sich das Nettoeinkommen im laufenden Jahr gegenüber dem Vorjahr (bzw. bei
              Selbständigen gegenüber der letzten Festsetzung) um mehr als 10 % verändert, ist dies
              unter Vorlage entsprechender Nachweise unverzüglich anzuzeigen.
            </p>
          </div>
        </div>

        <!-- Grußformel -->
        <div class="closing">
          <p class="body-text">Mit freundlichen Grüßen</p>
          <div class="closing__signature">Der Vorstand der Kita Knirpsenstadt e.&thinsp;V.</div>
        </div>

        <!-- Fußzeile -->
        <footer class="footer">
          <p class="footer__line">
            <span class="footer__label">Verein</span>
            Kita Knirpsenstadt e.&thinsp;V. &middot; Ahornallee 27 &middot; 16341 Panketal &middot;
            Vereinsregister VR 4217, Amtsgericht Frankfurt (Oder)
          </p>
          <p class="footer__line">
            <span class="footer__label">Vorstand</span>
            André Rüger (1. Vorsitzender) &middot; Sarah Thränhardt (2. Vorsitzende / Bauliches) &middot;
            Marcus Rehaag (Finanzen) &middot; Stefan Remer (Elternarbeit) &middot;
            Samantha Lahl (Schriftführerin) &middot; Dennis Braak (Personal)
          </p>
          <p class="footer__line">
            <span class="footer__label">Bank</span>
            Knirpsenstadt e.&thinsp;V. &middot; IBAN DE33 3702 0500 0003 3214 00 &middot;
            BIC BFSWDE33XXX &middot; Bank für Sozialwirtschaft AG
          </p>
          <p class="footer__legal">
            Rechtlich verbindliche Aussagen für den Verein trifft allein der Vorstand.
          </p>
        </footer>

      </div>
    </div>
  </div>
</template>

<style scoped src="./EinstufungPDF.css"></style>
