import { ref } from 'vue'
import { hasPermission } from './rbacAccess.js'

// The last GetStatus response; components read it reactively through useAppStatus().
const appStatus = ref(null)

let inFlight = null

function applyAppStatusToWindow(st) {
	window.userRbacPermissions = st?.rbacPermissions || []
	window.userRbacIsSuperuser = Boolean(st?.rbacIsSuperuser)
}

// fetchAppStatus returns the cached status, sharing one in-flight GetStatus per page load.
// Pass { force: true } after login and logout.
export async function fetchAppStatus({ force = false } = {}) {
	if (!force && appStatus.value) {
		return appStatus.value
	}
	if (!force && inFlight) {
		return inFlight
	}

	inFlight = window.client.getStatus({})
		.then((st) => {
			appStatus.value = st
			applyAppStatusToWindow(st)
			return st
		})
		.finally(() => {
			inFlight = null
		})

	return inFlight
}

export function useAppStatus() {
	return {
		appStatus,
		hasPermission: (name) => hasPermission(appStatus.value, name),
	}
}

// errorMessage turns a Connect error into something to show a user.
export function errorMessage(err, fallback = 'Something went wrong') {
	if (!err) {
		return fallback
	}
	return err.rawMessage || err.message || fallback
}
