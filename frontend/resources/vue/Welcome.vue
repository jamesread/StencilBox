<template>
	<section>
		<h1>Welcome to StencilBox!</h1>
		<p>Use the sidebar to navigate to the different sections of the application.</p>


		<p>StencilBox can operate in various modes, depending on your workflow;</p>

		<table>
			<thead>
				<tr>
					<th>Mode</th>
					<th>Config Files</th>
					<th>Web UI</th>
					<th>Use case</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Immutable filesystem, Read-Only UI</td>
					<td>
						<p>Read only</p>
					</td>
					<td>
						<p>Read only</p>
					</td>
					<td>
						<p>Pure GitOps</p>
					</td>
				</tr>
				<tr>
					<td>Immutable filesystem, UI with actions</td>
					<td>
						<p>Read only</p>
					</td>
					<td>
						<p>Actions: Start build, etc</p>
					</td>
					<td>
						<p>GitOps with actions</p>
					</td>
				</tr>
				<tr>
					<td>Mutable filesystem, UI with actions and editing</td>
					<td>
						<p>Editable</p>
					</td>
					<td>
						<p>Actions: Start build, edit files, etc</p>
					</td>
					<td>
						<p>Web UI focussed workflow, optional GitOps externally</p>
					</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>
