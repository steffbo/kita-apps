<script setup lang="ts">
import { MailPlus, ShieldCheck } from 'lucide-vue-next';
</script>

<template>
  <section class="mx-auto max-w-7xl px-5 py-8 sm:px-8 lg:px-10 lg:py-10">
    <div class="mb-8">
      <p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">Administration</p>
      <h1 class="mt-3 text-3xl font-semibold tracking-normal sm:text-4xl">Einladungen und Rollen</h1>
    </div>

    <div class="grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
      <form class="rounded-lg border bg-surface p-5 shadow-soft">
        <div class="mb-5 flex items-center justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold">Einladung</h2>
            <p class="mt-1 text-sm text-muted-foreground">Link für Passwort-Onboarding.</p>
          </div>
          <MailPlus class="h-5 w-5 text-primary" :stroke-width="1.8" />
        </div>
        <div class="grid gap-4">
          <label class="field">
            <span class="field-label">E-Mail</span>
            <input class="field-input" type="email" placeholder="name@example.org" />
          </label>
          <label class="field">
            <span class="field-label">Rolle</span>
            <select class="field-input">
              <option>PARENT</option>
              <option>TEAM</option>
              <option>LEITUNG</option>
              <option>BEITRAG</option>
              <option>VORSTAND</option>
              <option>ADMIN</option>
            </select>
          </label>
        </div>
        <button class="primary-button mt-5 w-full" type="button">Einladung vorbereiten</button>
      </form>

      <div class="rounded-lg border bg-surface p-5 shadow-soft">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-lg font-semibold">Offene Einladungen</h2>
          <ShieldCheck class="h-5 w-5 text-muted-foreground" :stroke-width="1.8" />
        </div>
        <div class="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed bg-background px-5 text-center">
          <p class="text-sm font-semibold">Keine offenen Einladungen geladen</p>
          <p class="mt-2 max-w-[42ch] text-sm leading-6 text-muted-foreground">
            Die Liste wird mit dem Invite-Endpunkt verbunden.
          </p>
        </div>
      </div>
    </div>
  </section>
</template>
