<script setup lang="ts">
import { computed, ref } from 'vue';
import type { Einstufung, Child } from '@/api/types';
import { FileDown, Loader2 } from 'lucide-vue-next';

const props = defineProps<{
  einstufung: Einstufung;
}>();

const isGenerating = ref(false);
const pdfContainer = ref<HTMLElement | null>(null);

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

const feeColumns = computed<FeeColumn[]>(() => {
  const e = props.einstufung;
  const validFrom = new Date(e.validFrom);
  const startMonth = validFrom.getMonth(); // 0-based
  const startYear = validFrom.getFullYear();

  const cols: FeeColumn[] = [];

  const formatMonth = (month: number, year: number) => {
    return new Date(year, month).toLocaleString('de-DE', { month: 'short', year: '2-digit' });
  };

  const formatCareType = (ct: string) => {
    if (ct === 'krippe') return 'Krippe';
    if (ct === 'kindergarten') return 'Kindergarten';
    return ct.charAt(0).toUpperCase() + ct.slice(1);
  };

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

const entryDateFormatted = computed(() => {
  if (!child.value?.entryDate) return '—';
  return new Date(child.value.entryDate).toLocaleDateString('de-DE');
});

const birthDateFormatted = computed(() => {
  if (!child.value?.birthDate) return '—';
  return new Date(child.value.birthDate).toLocaleDateString('de-DE');
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

async function generatePdf() {
  if (!pdfContainer.value) return;
  isGenerating.value = true;

  try {
    const html2pdf = (await import('html2pdf.js')).default;
    const opt = {
      margin: [10, 12, 15, 12],
      filename: `Einstufung_${einstufungYear.value}_${childName.value.replace(/\s/g, '_')}.pdf`,
      image: { type: 'jpeg', quality: 0.98 },
      html2canvas: { scale: 2, useCORS: true },
      jsPDF: { unit: 'mm', format: 'a4', orientation: 'portrait' as const },
    };

    await html2pdf().set(opt).from(pdfContainer.value).save();
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
      PDF herunterladen
    </button>

    <!-- Hidden PDF content (rendered off-screen for html2pdf) -->
    <div class="fixed left-[-9999px] top-0">
      <div ref="pdfContainer" class="page">

        <!-- page-header -->
        <div class="page-header">
          <div class="page-header__sender">
            Elternverein Kita Knirpsenstadt e.V. &middot; Ahornallee 27 &middot; 16341 Panketal
          </div>
          <div class="page-header__sub">Der Vorstand der Kita</div>
          <div class="page-header__rule"></div>
        </div>

        <!-- title -->
        <div class="title">
          Einstufung Elternbeiträge {{ einstufungYear }}
        </div>

        <!-- info-grid -->
        <div class="info-grid">
          <div class="info-grid__name">{{ childName }}</div>
          <div class="info-grid__fields">
            <div class="info-item">
              <div class="info-item__label">Geburtsdatum</div>
              <div class="info-item__value">{{ birthDateFormatted }}</div>
            </div>
            <div class="info-item">
              <div class="info-item__label">Besucht seit</div>
              <div class="info-item__value">{{ entryDateFormatted }}</div>
            </div>
            <div class="info-item">
              <div class="info-item__label">Mitgliedsnummer</div>
              <div class="info-item__value">{{ memberNumber }}</div>
            </div>
            <div class="info-item info-item--wide">
              <div class="info-item__label">Einrichtung</div>
              <div class="info-item__value">Kita Knirpsenstadt e.V., Ahornallee 27, 16341 Panketal</div>
            </div>
          </div>
        </div>

        <!-- section: Rechtstext -->
        <section class="section">
          <p class="body-text">
            Nach § 17 des Kindertagesstättengesetzes haben die Erziehungsberechtigten Beiträge zur
            Inanspruchnahme eines Platzes in der Kindertagesstätte zu entrichten. Dieser monatliche
            Elternbeitrag wird in Verbindung mit der Elternbeitragsordnung des Trägers ermittelt.
            Die Kindertagesstätte „Knirpsenstadt" in 16341 Panketal, Ahornallee 27 befindet sich in
            freier Trägerschaft des „Knirpsenstadt e.V. Panketal".
          </p>
          <p class="body-text">
            Berechnet wird nach wirtschaftlicher Leistungsfähigkeit (Nettoeinkommen im Jahr), dem
            Alter des Kindes und der beanspruchten Betreuungszeit. Eine Ermäßigung des Elternbeitrages
            wird auch nach der Anzahl der unterhaltspflichtigen Kinder gewährt (jedoch nicht nach dem
            Brandenburg Entlastungspaket).
          </p>
        </section>

        <!-- notice-box: Einstufungsgrundlage -->
        <div class="notice-box">
          <div class="notice-box__label">Grundlage der Einstufung</div>
          <div class="notice-box__text">{{ feeRuleText }}</div>
        </div>

        <!-- section: Beitragsübersicht -->
        <section class="section">
          <div class="section__heading">Monatliche Beiträge</div>

          <table class="fee-table">
            <thead>
              <tr>
                <th class="fee-table__col-label"></th>
                <th
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__col-month"
                  :class="{ 'fee-table__col-month--first': idx === 0 }"
                >
                  <div class="fee-table__month-name">{{ col.label }}</div>
                  <div class="fee-table__month-sub">{{ col.careType }} &middot; {{ col.careHours }} h/Woche</div>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr class="fee-table__row">
                <td class="fee-table__row-label">Platzgeld</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': idx === 0 }"
                >
                  {{ formatEur(col.childcareFee) }}
                </td>
              </tr>
              <tr class="fee-table__row">
                <td class="fee-table__row-label">Essensgeld</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': idx === 0 }"
                >
                  {{ formatEur(col.foodFee) }}
                </td>
              </tr>
              <tr v-if="feeColumns.some(c => c.membershipFee > 0)" class="fee-table__row fee-table__row--membership">
                <td class="fee-table__row-label fee-table__row-label--membership">Vereinsbeitrag (jährlich)</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount fee-table__amount--membership"
                  :class="{ 'fee-table__amount--primary': idx === 0 }"
                >
                  {{ col.membershipFee > 0 ? formatEur(col.membershipFee) : '—' }}
                </td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- payment-note -->
        <div class="payment-note">
          <div class="payment-note__marker">Hinweis</div>
          <div class="payment-note__text">
            Bitte gleicht die Beträge für Mitgliedschaft, Betreuung und Essensgeld in
            <strong class="payment-note__emphasis">getrennten</strong> Zahlungen unter Angabe des
            Namens und der Mitgliedsnummer aus.
          </div>
        </div>

        <!-- section: Zahlungsbedingungen & Änderungspflicht -->
        <section class="section">
          <div class="section__heading">Zahlungsbedingungen</div>
          <p class="body-text">
            Der monatliche Beitrag wird am 5. eines jeden Monats fällig. Beiträge, die einen Monat
            in Verzug sind, werden zusätzlich mit einer Mahngebühr von 10,00 € erhoben.
          </p>
          <p class="body-text">
            Der Vereinsbeitrag (derzeit 30,00 €) ist jährlich zu zahlen: Bei Vertragsbeginn sofort,
            ansonsten bis spätestens Ende des ersten Quartals. Nach Fristablauf wird ein Mahngeld
            von 5,00 € erhoben.
          </p>
          <div class="section__heading section__heading--sub">Änderungspflicht</div>
          <p class="body-text">
            Wenn sich das Nettoeinkommen im laufenden Jahr gegenüber dem Vorjahr (bzw. bei
            Selbständigen gegenüber der letzten Festsetzung) um mehr als 10 % verändert, ist dies
            unter Vorlage entsprechender Nachweise unverzüglich anzuzeigen.
          </p>
        </section>

        <!-- footer -->
        <footer class="footer">
          <div class="footer__register">
            Kita Knirpsenstadt e.V. &middot; Vereinsregister VR 4217 beim Amtsgericht Frankfurt (Oder)
          </div>
          <div class="footer__columns">
            <div class="footer__col">
              <div class="footer__col-heading">Vorstandsmitglieder</div>
              André Rüger (1. Vorsitzender)<br>
              Sarah Thielandt (2. Vorsitzende / Bauliches)<br>
              Marcus Rehberg (Kassenwart)<br>
              Stefan Remer (Elternarbeit)<br>
              Samantha Lahl (Schriftführer)<br>
              Dennis Braak (Personal)
            </div>
            <div class="footer__col">
              <div class="footer__col-heading">Bankverbindung</div>
              Knirpsenstadt e. V.<br>
              IBAN: DE53 3702 0500 0003 3714 00<br>
              BIC: BFSWDE33XXX<br>
              Bank für Sozialwirtschaft AG
            </div>
          </div>
          <div class="footer__legal">
            Rechtlich verbindliche Aussagen für den Verein trifft allein der Vorstand.
          </div>
        </footer>

      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Design Tokens ─────────────────────────────────────────────────────────── */
.page {
  width: 186mm;
  font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Arial, sans-serif;
  font-size: 10.5px;
  color: #1c1c1c;
  line-height: 1.55;
  background: #ffffff;
  padding: 0;
}

/* ── Page Header ───────────────────────────────────────────────────────────── */
.page-header {
  margin-bottom: 22px;
}

.page-header__sender {
  font-size: 8.5px;
  color: #888;
  letter-spacing: 0.2px;
  margin-bottom: 3px;
}

.page-header__sub {
  font-size: 9.5px;
  color: #555;
  font-weight: 500;
}

.page-header__rule {
  margin-top: 10px;
  height: 1px;
  background: #d1d5db;
}

/* ── Title ─────────────────────────────────────────────────────────────────── */
.title {
  font-size: 18px;
  font-weight: 700;
  color: #1c6a38;
  letter-spacing: -0.2px;
  margin-bottom: 18px;
  padding-bottom: 10px;
  border-bottom: 2px solid #1c6a38;
}

/* ── Info Grid (Stammdaten) ────────────────────────────────────────────────── */
.info-grid {
  border: 1px solid #d1d5db;
  border-left: 3px solid #1c6a38;
  padding: 12px 14px;
  margin-bottom: 18px;
  background: #fafafa;
}

.info-grid__name {
  font-size: 12.5px;
  font-weight: 700;
  color: #1c1c1c;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e5e7eb;
}

.info-grid__fields {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
}

.info-item {
  flex: 1;
  min-width: 110px;
  padding-right: 12px;
}

.info-item--wide {
  flex: 2;
  min-width: 160px;
}

.info-item__label {
  font-size: 8.5px;
  color: #777;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  margin-bottom: 2px;
}

.info-item__value {
  font-size: 10px;
  font-weight: 600;
  color: #1c1c1c;
  line-height: 1.35;
}

/* ── Sections ──────────────────────────────────────────────────────────────── */
.section {
  margin-bottom: 16px;
}

.section__heading {
  font-size: 11px;
  font-weight: 700;
  color: #1c1c1c;
  margin-bottom: 8px;
  padding-bottom: 4px;
  border-bottom: 1.5px solid #1c6a38;
}

.section__heading--sub {
  margin-top: 10px;
  border-bottom-color: #d1d5db;
}

/* ── Body Text ─────────────────────────────────────────────────────────────── */
.body-text {
  font-size: 9.5px;
  color: #3d3d3d;
  line-height: 1.65;
  text-align: justify;
  margin-bottom: 8px;
}

.body-text:last-child {
  margin-bottom: 0;
}

/* ── Notice Box (Einstufungsgrundlage) ─────────────────────────────────────── */
.notice-box {
  border: 1px solid #a7c9b2;
  border-left: 3px solid #1c6a38;
  background: #f4fbf6;
  padding: 10px 14px;
  margin-bottom: 16px;
}

.notice-box__label {
  font-size: 8px;
  font-weight: 700;
  color: #1c6a38;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 4px;
}

.notice-box__text {
  font-size: 9.5px;
  color: #1c3a26;
  line-height: 1.6;
  font-weight: 500;
}

/* ── Fee Table ─────────────────────────────────────────────────────────────── */
.fee-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 0;
  font-size: 9.5px;
}

.fee-table th,
.fee-table td {
  padding: 6px 8px;
  text-align: right;
}

.fee-table th:first-child,
.fee-table td:first-child {
  text-align: left;
  padding-left: 0;
}

.fee-table thead tr {
  border-bottom: 1.5px solid #1c6a38;
}

.fee-table__col-label {
  width: 40%;
}

.fee-table__col-month {
  color: #3d3d3d;
  font-weight: 400;
}

.fee-table__col-month--first {
  color: #1c6a38;
}

.fee-table__month-name {
  font-size: 10px;
  font-weight: 700;
  margin-bottom: 1px;
}

.fee-table__month-sub {
  font-size: 8px;
  font-weight: 400;
  color: #777;
}

.fee-table__row td {
  border-bottom: 1px solid #ececec;
  color: #1c1c1c;
}

.fee-table__row-label {
  color: #555;
  font-size: 9.5px;
}

.fee-table__amount {
  font-size: 10px;
  font-weight: 600;
  color: #1c1c1c;
  white-space: nowrap;
}

.fee-table__amount--primary {
  color: #1c6a38;
}

.fee-table__row--membership td {
  border-top: 1px dashed #c3d9c9;
  border-bottom: none;
  padding-top: 8px;
}

.fee-table__row-label--membership {
  font-size: 9px;
  color: #444;
}

.fee-table__amount--membership {
  font-size: 9.5px;
  color: #444;
}

.fee-table__amount--membership.fee-table__amount--primary {
  color: #1c6a38;
}

/* ── Payment Note ──────────────────────────────────────────────────────────── */
.payment-note {
  display: flex;
  align-items: baseline;
  gap: 10px;
  background: #fffbf0;
  border: 1px solid #d4a72c;
  border-left: 3px solid #b8860b;
  padding: 9px 14px;
  margin-bottom: 16px;
}

.payment-note__marker {
  font-size: 7.5px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #b8860b;
  white-space: nowrap;
  flex-shrink: 0;
}

.payment-note__text {
  font-size: 9.5px;
  color: #3d2e00;
  line-height: 1.6;
}

.payment-note__emphasis {
  font-weight: 700;
  text-decoration: underline;
  text-decoration-color: #b8860b;
}

/* ── Footer ────────────────────────────────────────────────────────────────── */
.footer {
  border-top: 1px solid #c8c8c8;
  padding-top: 12px;
  margin-top: 20px;
}

.footer__register {
  font-size: 8.5px;
  color: #555;
  font-weight: 600;
  margin-bottom: 8px;
}

.footer__columns {
  display: flex;
  gap: 24px;
  font-size: 8px;
  color: #555;
  line-height: 1.55;
  margin-bottom: 8px;
}

.footer__col {
  flex: 1;
}

.footer__col-heading {
  font-size: 8px;
  font-weight: 700;
  color: #1c1c1c;
  margin-bottom: 3px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.footer__legal {
  font-size: 7.5px;
  color: #999;
  font-style: italic;
}
</style>
