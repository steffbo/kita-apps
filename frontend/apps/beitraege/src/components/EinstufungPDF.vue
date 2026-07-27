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

// Zeiträume (statt Einzelmonate) für die Beitragstabelle
interface FeePeriod {
  key: string;
  label: string; // z. B. "Aug. 2026 – Feb. 2027" oder "ab März 2027"
  careHours: number;
  careType: string; // "Krippe" oder "Kindergarten"
  childcareFee: number;
  foodFee: number;
  isPrevious: boolean; // bisheriger Beitrag vor einer Folge-Einstufung
  isCurrent: boolean; // aktuell gültiger Beitrag
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

function isSameMonth(left: Date, right: Date): boolean {
  return left.getUTCFullYear() === right.getUTCFullYear() && left.getUTCMonth() === right.getUTCMonth();
}

function formatCareType(ct: string) {
  if (ct === 'krippe') return 'Krippe';
  if (ct === 'kindergarten') return 'Kindergarten';
  return ct.charAt(0).toUpperCase() + ct.slice(1);
}

function formatMonthLabel(date: Date): string {
  return date.toLocaleString('de-DE', { month: 'short', year: 'numeric', timeZone: 'UTC' });
}

function formatPeriodLabel(from: Date, to: Date | null): string {
  if (!to) return `ab ${formatMonthLabel(from)}`;
  if (isSameMonth(from, to)) return formatMonthLabel(from);
  return `${formatMonthLabel(from)} \u2013 ${formatMonthLabel(to)}`;
}

// Monat, ab dem das Kind als Kindergartenkind gilt (erster voller Monat nach dem 3. Geburtstag)
const kindergartenFromMonth = computed<Date | null>(() => {
  if (!child.value || props.einstufung.careType !== 'krippe') return null;

  const birthDate = new Date(child.value.birthDate);
  const turnsThree = new Date(birthDate.getFullYear() + 3, birthDate.getMonth(), birthDate.getDate());
  let month = turnsThree.getMonth();
  let year = turnsThree.getFullYear();
  if (turnsThree.getDate() > 1) {
    month += 1;
    if (month > 11) {
      month = 0;
      year += 1;
    }
  }
  return new Date(Date.UTC(year, month, 1));
});

const feePeriods = computed<FeePeriod[]>(() => {
  const e = props.einstufung;
  const validFrom = parseMonthStart(e.effectiveFromMonth || e.validFrom) ?? new Date();
  const periods: FeePeriod[] = [];

  // Bisheriger Beitrag, wenn diese Einstufung eine Folge-Einstufung ist
  if (e.sourceEinstufungId && props.previousEinstufung) {
    const previous = props.previousEinstufung;
    const previousEnd = addUtcMonths(validFrom, -1);
    const previousStart = parseMonthStart(previous.effectiveFromMonth || previous.validFrom);

    if (previousStart && isSameOrAfterMonth(previousEnd, previousStart)) {
      const previousRow = previous.monthlyTable?.find((row) =>
        row.year === previousEnd.getUTCFullYear() && row.month === previousEnd.getUTCMonth() + 1
      );

      periods.push({
        key: 'previous',
        label: formatPeriodLabel(previousStart, previousEnd),
        careHours: previousRow?.careHoursPerWeek ?? previous.careHoursPerWeek,
        careType: previousRow?.careType ?? formatCareType(previous.careType),
        childcareFee: previousRow?.childcareFee ?? previous.monthlyChildcareFee,
        foodFee: previousRow?.foodFee ?? previous.monthlyFoodFee,
        isPrevious: true,
        isCurrent: false,
      });
    }
  }

  // Ende dieser Einstufung, falls bereits eine Folge-Einstufung existiert
  const closedUntil = parseMonthStart(e.validUntil);
  const kindergartenFrom = kindergartenFromMonth.value;
  const switchesToKindergarten =
    !!kindergartenFrom &&
    isSameOrAfterMonth(kindergartenFrom, addUtcMonths(validFrom, 1)) &&
    (!closedUntil || isSameOrAfterMonth(closedUntil, kindergartenFrom));

  const currentEnd = switchesToKindergarten
    ? addUtcMonths(kindergartenFrom as Date, -1)
    : closedUntil ?? null;

  periods.push({
    key: 'current',
    label: formatPeriodLabel(validFrom, currentEnd),
    careHours: e.careHoursPerWeek,
    careType: formatCareType(e.careType),
    childcareFee: e.monthlyChildcareFee,
    foodFee: e.monthlyFoodFee,
    isPrevious: false,
    isCurrent: true,
  });

  // Ab dem Wechsel in den Kindergarten entfällt das Platzgeld
  if (switchesToKindergarten && kindergartenFrom) {
    periods.push({
      key: 'kindergarten',
      label: formatPeriodLabel(kindergartenFrom, closedUntil ?? null),
      careHours: e.careHoursPerWeek,
      careType: 'Kindergarten',
      childcareFee: 0,
      foodFee: e.monthlyFoodFee,
      isPrevious: false,
      isCurrent: false,
    });
  }

  return periods;
});

const isFollowUp = computed(() => !!props.einstufung.sourceEinstufungId);
const membershipFee = computed(() => props.einstufung.annualMembershipFee);
const showMembershipNote = computed(() => isFollowUp.value || membershipFee.value > 0);
const membershipDueMonth = computed(() => {
  const start = parseMonthStart(props.einstufung.effectiveFromMonth || props.einstufung.validFrom);
  return start ? formatMonthLabel(start) : '—';
});

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
                <th class="fee-table__col-period">Zeitraum</th>
                <th class="fee-table__col-area">Bereich</th>
                <th class="fee-table__col-hours">Std./Woche</th>
                <th class="fee-table__col-amount">Platzgeld</th>
                <th class="fee-table__col-amount">Essensgeld</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="period in feePeriods"
                :key="period.key"
                :class="{
                  'fee-table__row--previous': period.isPrevious,
                  'fee-table__row--current': period.isCurrent,
                }"
              >
                <td class="fee-table__period">
                  {{ period.label }}
                  <span v-if="period.isPrevious" class="fee-table__period-note">(bisher)</span>
                </td>
                <td class="fee-table__text">{{ period.careType }}</td>
                <td class="fee-table__amount">{{ period.careHours }}</td>
                <td class="fee-table__amount">{{ formatEur(period.childcareFee) }}</td>
                <td class="fee-table__amount">{{ formatEur(period.foodFee) }}</td>
              </tr>
            </tbody>
          </table>
          <p v-if="showMembershipNote" class="fee-table__note">
            Vereinsbeitrag: {{ formatEur(membershipFee) }} jährlich
            <template v-if="isFollowUp">
              &ndash; für dieses Jahr bereits bezahlt, in den oben genannten Monatsbeträgen nicht enthalten.
            </template>
            <template v-else>
              &ndash; einmalig fällig mit dem ersten Beitrag im {{ membershipDueMonth }}, in den
              Monatsbeträgen nicht enthalten.
            </template>
          </p>
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
