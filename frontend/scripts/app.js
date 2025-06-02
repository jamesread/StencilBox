import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { StencilBoxApiService } from './proto/StencilBox/clientapi/v1/clientapi_pb'

export function init() {
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

  if (status.buildConfigs.length === 0) {
    let section = document.createElement('section');
    section.innerText = 'No build configurations found yet, please write one!';

    document.getElementsByTagName('main')[0].appendChild(section);
  } else {
    for (const bc of status.buildConfigs) {
      createBuildConfigSection(bc)
    }
  }

  renderTemplateList(status.templates);

  let homeLink = createHeaderLink('Build Configs', 'build-config');
  createHeaderLink('Templates', 'template')

  homeLink.dispatchEvent(new MouseEvent('click'));
}

function createHeaderLink(name, templateClass) {
  let link = document.createElement('a');
  link.innerText = name;
  link.href = '#'
  link.onclick = (e) => {
    document.querySelectorAll('#header-links li a').forEach(a => {
      a.classList.remove('active');
    });
    e.target.classList.add('active');
    showSectionsWithClass(templateClass)
  }

  let li = document.createElement('li');
  li.appendChild(link);

  document.getElementById('header-links').appendChild(li);

  return link;
}

function showSectionsWithClass(className) {
  for (const section of document.querySelectorAll('section')) {
    if (section.classList.contains(className)) {
      section.hidden = false;
    } else {
      section.hidden = true;
    }
  }
}

function renderTemplateList(templates) {
  // sort templates by name
  templates.sort((a, b) => a.name.localeCompare(b.name));

  for (const template of templates) {
    let row = document.createElement('tr');

    let nameCell = document.createElement('td');
    nameCell.innerText = template.name;
    row.appendChild(nameCell);

    let sourceCell = document.createElement('td');
    sourceCell.innerText = template.source;
    row.appendChild(sourceCell);

    let statusCell = document.createElement('td');
    statusCell.innerText = template.status;
    statusCell.classList.add('good');
    row.appendChild(statusCell);

    document.getElementById('template-table-rows').appendChild(row);
  }
}

function createBuildConfigSection(buildConfig) {
  console.log('Creating build config section for:', buildConfig);

  let tpl = document.getElementById('build-config-template').content.cloneNode(true);

  tpl.querySelector('section').setAttribute('title', buildConfig.name);
  tpl.querySelector('.build-config-name').innerText = buildConfig.name;
  tpl.querySelector('.build-template').innerText = buildConfig.template;

  tpl.querySelector('.start-build-button').onclick = () => {
    startBuild(buildConfig.name);
  }

  document.getElementsByTagName('main')[0].appendChild(tpl);
}

function startBuild(buildConfigName) {
  window.client.startBuild({
    'configName': buildConfigName
  })
    .then(response => {
      onBuildStarted(response);
    })
    .catch(error => {
      console.error('Error starting build:', error);
    });
}

function onBuildStarted(response) {
  console.log('Build started:', response);

  if (!response.found) {
    showBigError('Build config not found. Please check the configuration.');
    return;
  }

  let buildSection = document.querySelector(`section[title="${response.configName}"]`);

  if (!buildSection) {
    showBigError('Build section not found. Please refresh the page.');
  }

  buildSection.querySelector('.build-status').innerText = response.status;

  if (response.isError) {
    buildSection.querySelector('.build-status').classList.add('error');
    buildSection.querySelector('.build-status').classList.remove('good');
  } else {
    buildSection.querySelector('.build-status').classList.remove('error');
    buildSection.querySelector('.build-status').classList.add('good');
  }

  let l = window.location;
  let a = document.createElement('a');
  a.href = l.origin + '/' + response.relativePath

  a.innerText = 'LINK';

  let urlContainer = buildSection.querySelector('.build-url')
  urlContainer.innerHTML = '';
  urlContainer.appendChild(a)

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
