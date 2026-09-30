import { createApp } from 'vue';
import { createPinia } from 'pinia';

import App from './App.vue';
import router from './router';
import { installDialogEscape } from './utils/dialogEscape';
import { installGermanValidation } from './utils/germanValidation';
import '@fontsource-variable/nunito';

import './assets/main.css';

const app = createApp(App);

app.use(createPinia());
app.use(router);

app.mount('#app');
installDialogEscape();
installGermanValidation();
