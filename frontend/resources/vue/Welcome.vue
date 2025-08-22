<template>
	<Section title = "Welcome">
		<p><strong>Generate super speedy simple static sites, from sleek templates.</strong></p>

		<p>Here's a quick list of suggestions for things you can build with StencilBox:</p>
		<ul>
			<li>Build a "homepage", "service catalog" or "landing page" for links that you can easily update from YAML (and loads super duper quickly!)</li>
			<li>Build a quick "site switching sidebar" for your team, so they can easily find the right site.</li>
			<li>Build automated reports from data files, like a status page.</li>
			<li>Build pretty documentation out of YAML data files, like a FAQ, or make a config file easy to view.</li>
		</ul>
	</Section>

	<section>
		<h2>
			You will need to write YAML!
			<HugeiconsIcon :icon = "File01Icon" size = "24" />
		</h2>

		<p>StencilBox uses YAML to define build configurations, and data files. you won't find a "create" button in the StencilBox interface. The way that StencilBox works is that you write a YAML file that defines a build configuration, and a data file that contains the data for the build. Then you run the build, and StencilBox will use the build configuration to build the data file, and the template to build the output.</p>

		<a class = "button" href = "https://jamesread.github.io/StencilBox/getting-started.html">
			Getting Started docs
			<HugeiconsIcon :icon = "LinkSquare01Icon" size = "24" />
		</a>
	</section>
</template>

<script setup>
import { HugeiconsIcon } from '@hugeicons/vue';
import { File01Icon, LinkSquare01Icon } from '@hugeicons/core-free-icons';

import Section from 'picocrank/vue/components/Section.vue';
</script>
