import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { StencilBoxApiService } from './proto/StencilBox/clientapi/v1/clientapi_pb'

import { createApp } from 'vue'
import router from './router.js'

import App from '../resources/vue/App.vue'

export function init() {
  createApiClient();
  setupVue();
}

function setupVue() {
  const app = createApp(App);

  app.use(router);
  app.config.globalProperties.$client = window.client;
  app.mount('#app');
}

function createApiClient() {
	// Always same-origin, so the session cookie is sent. In development Vite proxies /api to the
	// service (see vite.config.js).
	window.transport = createConnectTransport({
		baseUrl: '/api/',
	})

	window.client = createClient(StencilBoxApiService, window.transport)
}


function onBuildStarted(response) {
}

function showBigError(message) {
  console.error('Big error:', message);

  let el = document.createElement('dialog');
  el.classList.add('critical')
  el.innerHTML = `
    <h2>Critical Error</h2>
    <p>${message}</p>
    <form method="dialog">
    <button type = "close">Close</button>
    </form>
  `;

  document.body.appendChild(el);
  el.showModal()
}
