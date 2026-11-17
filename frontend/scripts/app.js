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
