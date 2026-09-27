import { ref } from 'vue';

export type ThemeMode = 'system' | 'light' | 'dark';

const key = 'kita-theme';
let saved: string | null = null;
try { saved = localStorage.getItem(key); } catch { /* Privater Browsermodus ohne Speicherung. */ }
const mode = ref<ThemeMode>(saved === 'light' || saved === 'dark' ? saved : 'system');
const dark = ref(false);
const media = window.matchMedia('(prefers-color-scheme: dark)');

function applyTheme() {
  dark.value = mode.value === 'dark' || (mode.value === 'system' && media.matches);
  document.documentElement.classList.toggle('dark', dark.value);
}

function setTheme(value: ThemeMode) {
  mode.value = value;
  try { localStorage.setItem(key, value); } catch { /* Die Wahl gilt bis zum Neuladen. */ }
  applyTheme();
}

media.addEventListener('change', applyTheme);
applyTheme();

function useTheme() {
  return { mode, dark, setTheme };
}

export { useTheme };
