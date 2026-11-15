<template>
	<div class="yaml-editor" :class="{ invalid: Boolean(problem), readonly }">
		<div class="yaml-editor-body">
			<pre ref="gutter" class="yaml-editor-gutter" aria-hidden="true">{{ lineNumbers }}</pre>
			<textarea
				:id="id"
				ref="textarea"
				:value="modelValue"
				:rows="rows"
				:readonly="readonly"
				:aria-label="ariaLabel || undefined"
				:aria-invalid="problem ? 'true' : undefined"
				spellcheck="false"
				autocapitalize="off"
				autocomplete="off"
				wrap="off"
				@input="onInput"
				@keydown="onKeydown"
				@scroll="syncScroll"
			/>
		</div>
		<p v-if="problem" class="yaml-editor-status bad" role="status">
			<span v-if="problem.line">Line {{ problem.line }}<span v-if="problem.col">, column {{ problem.col }}</span>: </span>{{ problem.message }}
		</p>
		<p v-else-if="!readonly" class="yaml-editor-status good" role="status">Valid YAML</p>
	</div>
</template>

<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import { parseDocument } from 'yaml'

// A YAML code textarea: line numbers, two-space indentation with Tab, and live validation. The
// server validates again on save.

const props = defineProps({
	modelValue: { type: String, default: '' },
	rows: { type: Number, default: 20 },
	readonly: { type: Boolean, default: false },
	id: { type: String, default: undefined },
	ariaLabel: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'validity'])

const INDENT = '  '
const textarea = ref(null)
const gutter = ref(null)

const lineNumbers = computed(() => {
	const count = props.modelValue.split('\n').length
	return Array.from({ length: Math.max(count, props.rows) }, (_, i) => i + 1).join('\n')
})

// The first parse error, or null when the text is valid YAML.
const problem = computed(() => {
	const doc = parseDocument(props.modelValue, { prettyErrors: true })
	const err = doc.errors[0]
	if (!err) {
		return null
	}
	const pos = err.linePos?.[0]
	return {
		line: pos?.line,
		col: pos?.col,
		// The first line is the explanation; the rest is a source excerpt.
		message: err.message.split('\n')[0].replace(/ at line \d+, column \d+:?$/, ''),
	}
})

watch(problem, (p) => emit('validity', !p), { immediate: true })

function onInput(event) {
	emit('update:modelValue', event.target.value)
}

function syncScroll() {
	if (gutter.value && textarea.value) {
		gutter.value.scrollTop = textarea.value.scrollTop
	}
}

// replaceSelection edits through execCommand where it exists, so the browser's undo still works.
function replaceSelection(el, text, selectStart, selectEnd) {
	el.focus()
	const done = typeof document.execCommand === 'function' && document.execCommand('insertText', false, text)
	if (!done) {
		el.setRangeText(text, el.selectionStart, el.selectionEnd, 'end')
	}
	emit('update:modelValue', el.value)
	if (selectStart !== undefined) {
		nextTick(() => el.setSelectionRange(selectStart, selectEnd))
	}
}

function onKeydown(event) {
	if (props.readonly || event.key !== 'Tab' || event.ctrlKey || event.metaKey || event.altKey) {
		return
	}
	const el = event.target
	const { selectionStart: start, selectionEnd: end, value } = el

	// A plain Tab with no selection indents; with a selection, (Shift+)Tab (un)indents the lines.
	if (start === end && !event.shiftKey) {
		event.preventDefault()
		replaceSelection(el, INDENT)
		return
	}

	event.preventDefault()
	const lineStart = value.lastIndexOf('\n', start - 1) + 1
	const lineEnd = end > start && value[end - 1] === '\n' ? end - 1 : end
	const block = value.slice(lineStart, lineEnd)
	const lines = block.split('\n')
	const changed = event.shiftKey
		? lines.map((l) => l.replace(new RegExp('^ {1,' + INDENT.length + '}'), ''))
		: lines.map((l) => INDENT + l)
	const replacement = changed.join('\n')

	el.setSelectionRange(lineStart, lineEnd)
	replaceSelection(el, replacement, lineStart, lineStart + replacement.length)
}

defineExpose({ focus: () => textarea.value?.focus() })
</script>

<style scoped>
.yaml-editor-body {
	display: grid;
	grid-template-columns: auto 1fr;
	border: 1px solid color-mix(in srgb, currentColor 25%, transparent);
	border-radius: 0.4em;
	overflow: hidden;
}

.yaml-editor.invalid .yaml-editor-body {
	border-color: #c0392b;
}

.yaml-editor-gutter,
.yaml-editor textarea {
	font-family: ui-monospace, 'SFMono-Regular', Menlo, Consolas, 'Liberation Mono', monospace;
	font-size: 0.875rem;
	line-height: 1.5;
	padding: 0.5em 0.75em;
	margin: 0;
	tab-size: 2;
}

.yaml-editor-gutter {
	overflow: hidden;
	text-align: right;
	user-select: none;
	opacity: 0.5;
	border-right: 1px solid color-mix(in srgb, currentColor 15%, transparent);
	background: color-mix(in srgb, currentColor 4%, transparent);
	min-width: 3.5em;
}

.yaml-editor textarea {
	width: 100%;
	min-width: 0;
	border: none;
	border-radius: 0;
	resize: vertical;
	white-space: pre;
	overflow: auto;
	box-shadow: none;
}

.yaml-editor textarea:focus {
	outline: 2px solid color-mix(in srgb, currentColor 30%, transparent);
	outline-offset: -2px;
}

.yaml-editor-status {
	margin: 0.35em 0 0;
	font-size: 0.875rem;
}

.yaml-editor-status.good {
	opacity: 0.7;
}
</style>
