<template>
	<section>
		<SectionHeader title = "Welcome to StencilBox!" subtitle = "Use the sidebar to navigate to the different sections of the app." />
	</section>

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
</script>
