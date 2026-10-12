import type { Einstufung } from '@/api/types';
import { formatEuroAmount, formatMonthYear } from '@/utils/format';

const BASIS_LABEL: Record<Einstufung['incomeBasis'], string> = {
  INCOME: 'nach eurem Einkommen',
  HIGHEST_RATE: 'Höchstsatz',
  NOT_REQUIRED: 'ohne Einkommensnachweis (Kindergarten)',
};

function careTypeLabel(careType: string): string {
  return careType === 'kindergarten' ? 'Kindergarten' : 'Krippe';
}

/** What changed from `previous` to `current`, one line per fee-relevant attribute. */
export function einstufungChanges(current: Einstufung, previous: Einstufung): string[] {
  const changes: string[] = [];
  if (current.careType !== previous.careType) {
    changes.push(`Betreuungsbereich: bisher ${careTypeLabel(previous.careType)}, jetzt ${careTypeLabel(current.careType)}`);
  }
  if (current.careHoursPerWeek !== previous.careHoursPerWeek) {
    changes.push(`Betreuungszeit: bisher ${previous.careHoursPerWeek}, jetzt ${current.careHoursPerWeek} Wochenstunden`);
  }
  if (current.incomeBasis !== previous.incomeBasis) {
    changes.push(`Grundlage: bisher ${BASIS_LABEL[previous.incomeBasis]}, jetzt ${BASIS_LABEL[current.incomeBasis]}`);
  } else if (
    current.incomeBasis === 'INCOME'
    && Math.round((current.annualNetIncome ?? 0) * 100) !== Math.round((previous.annualNetIncome ?? 0) * 100)
  ) {
    changes.push(`Jahresnettoeinkommen: bisher ${formatEuroAmount(previous.annualNetIncome ?? 0, 'Euro')}, jetzt ${
      formatEuroAmount(current.annualNetIncome ?? 0, 'Euro')
    }`);
  }
  if (current.childrenCount !== previous.childrenCount) {
    changes.push(`Anzahl unterhaltspflichtiger Kinder: bisher ${previous.childrenCount}, jetzt ${current.childrenCount}`);
  }
  return changes;
}

/** Whether the Einstufung starts with the child's admission to the Kita. */
function startsWithAdmission(current: Einstufung, entryDate?: string): boolean {
  return !!entryDate && entryDate.slice(0, 7) >= current.effectiveFromMonth.slice(0, 7);
}

/**
 * Opening paragraph of the Einstufung email. A new child is welcomed; for a child
 * already in the Kita the text names what changed against the previous Einstufung
 * and that this is why a new one was needed.
 */
export function einstufungEmailIntro(current: Einstufung, previous: Einstufung | null, childFirstName: string, entryDate?: string): string {
  const evidenceThanks = current.incomeBasis === 'INCOME' ? '\n\nDanke für eure Unterlagen.' : '';
  if (!previous && startsWithAdmission(current, entryDate)) {
    return `willkommen bei uns in der Kita!${evidenceThanks}`;
  }

  const month = formatMonthYear(Number(current.effectiveFromMonth.slice(0, 4)), Number(current.effectiveFromMonth.slice(5, 7)));
  const child = childFirstName || 'euer Kind';
  if (!previous) {
    return `für ${child} haben wir die Elternbeiträge ab ${month} neu eingestuft.${evidenceThanks}`;
  }

  const changes = einstufungChanges(current, previous);
  const lines: string[] = [];
  if (changes.length) {
    lines.push(`für ${child} hat sich etwas geändert, deshalb ist ab ${month} eine neue Einstufung nötig:`);
    lines.push(changes.map((c) => `- ${c}`).join('\n'));
  } else {
    lines.push(`für ${child} haben wir die Elternbeiträge ab ${month} neu eingestuft.`);
  }
  if (Math.round(current.monthlyChildcareFee * 100) !== Math.round(previous.monthlyChildcareFee * 100)) {
    lines.push(`Das Platzgeld beträgt damit ${formatEuroAmount(current.monthlyChildcareFee, 'Euro')} statt bisher ${
      formatEuroAmount(previous.monthlyChildcareFee, 'Euro')
    }.`);
  } else {
    lines.push('Das Platzgeld bleibt gleich.');
  }
  return lines.join('\n\n') + evidenceThanks;
}
