import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { StencilBoxApiService } from './proto/StencilBox/clientapi/v1/clientapi_pb'

export function init() {
  console.log("App initialized");

  createApiClient();
  setupApi();
}

function createApiClient() {
	let baseUrl = '/api/'

	if (window.location.hostname.includes('localhost')) {
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

  document.getElementById('start-build').onclick = async () => {
    startBuild();
  }
}

function startBuild() {
  window.client.startBuild({})
    .then(response => {
      onBuildStarted(response);
    })
    .catch(error => {
      console.error('Error starting build:', error);
    });
}

function onBuildStarted(response) {
  console.log('Build started:', response);

  document.getElementById('last-built').innerText = 'Build started successfully!';
}
