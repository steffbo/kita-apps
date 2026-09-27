<script setup lang="ts">
import type { EmailLog } from '@/api/types';
import { X } from 'lucide-vue-next';
import { formatDateTime } from '@/utils/format';
import { formatEmailType } from '@/utils/reminders';

defineProps<{ log: EmailLog }>();
defineEmits<{ close: [] }>();
</script>

<template>
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
    @click.self="$emit('close')"
  >
    <div class="bg-card rounded-xl shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col">
      <div class="flex items-start justify-between gap-4 p-5 border-b">
        <div class="min-w-0">
          <h3 class="text-lg font-semibold text-foreground">Gesendete E-Mail</h3>
          <p class="text-sm text-muted-foreground truncate">{{ log.subject }}</p>
        </div>
        <button
          type="button"
          class="rounded-lg p-2 text-muted-foreground hover:bg-accent hover:text-foreground"
          aria-label="Modal schließen"
          @click="$emit('close')"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <div class="overflow-y-auto p-5">
        <dl class="grid gap-4 text-sm sm:grid-cols-2">
          <div>
            <dt class="font-medium text-muted-foreground">Zeitpunkt</dt>
            <dd class="mt-1 text-foreground">{{ formatDateTime(log.sentAt ?? '') }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Typ</dt>
            <dd class="mt-1 text-foreground">{{ formatEmailType(log.emailType ?? '') }}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Empfänger</dt>
            <dd class="mt-1 break-all text-foreground">{{ log.toEmail }}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Betreff</dt>
            <dd class="mt-1 text-foreground">{{ log.subject }}</dd>
          </div>
        </dl>

        <pre class="mt-5 max-h-[55vh] overflow-auto whitespace-pre-wrap rounded-lg border bg-muted p-4 text-sm leading-6 text-foreground">{{ log.body || '-' }}</pre>
      </div>

      <div class="flex justify-end p-5 border-t">
        <button
          type="button"
          class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-accent"
          @click="$emit('close')"
        >
          Schließen
        </button>
      </div>
    </div>
  </div>
</template>
