import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { StencilBoxApiService } from './proto/StencilBox/clientapi/v1/clientapi_pb'

import { createApp } from 'vue';

import App from '../resources/vue/App.vue';

export function init() {
  setupVue();

  createApiClient();
  setupApi();
}

function setupVue() {
  const app = createApp(App);

  app.config.globalProperties.$client = window.client;
  app.mount('#app');
}

function createApiClient() {
	let baseUrl = '/api/'

	if (window.location.hostname.includes('localhost') && window.location.port === '5173') {
		baseUrl = 'http://localhost:8080/api/'
	}

	window.transport = createConnectTransport({
		baseUrl: baseUrl,
	})

	window.client = createClient(StencilBoxApiService, window.transport)
}

async function setupApi() {
  const status = await window.client.init();

  document.getElementById('current-version').innerText = 'Version: ' + status.version;
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
