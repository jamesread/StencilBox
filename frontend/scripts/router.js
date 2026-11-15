import { createRouter, createWebHistory } from 'vue-router'

import Welcome from '../resources/vue/Welcome.vue'
import BuildConfigList from '../resources/vue/BuildConfigList.vue'
import BuildConfigView from '../resources/vue/BuildConfigView.vue'
import TemplateList from '../resources/vue/TemplateList.vue'
import TemplateView from '../resources/vue/TemplateView.vue'
import TemplateAdd from '../resources/vue/TemplateAdd.vue'
import SystemDetails from '../resources/vue/SystemDetails.vue'
import DataFileList from '../resources/vue/DataFileList.vue'
import DataFileView from '../resources/vue/DataFileView.vue'
import ControlPanel from '../resources/vue/ControlPanel.vue'
import IamHub from '../resources/vue/iam/IamHub.vue'
import IamUsers from '../resources/vue/iam/IamUsers.vue'
import IamUserDetails from '../resources/vue/iam/IamUserDetails.vue'
import IamUserGroups from '../resources/vue/iam/IamUserGroups.vue'
import IamUserGroupDetails from '../resources/vue/iam/IamUserGroupDetails.vue'
import IamRoles from '../resources/vue/iam/IamRoles.vue'
import IamRoleDetails from '../resources/vue/iam/IamRoleDetails.vue'
import IamPermissions from '../resources/vue/iam/IamPermissions.vue'
import MyAccount from '../resources/vue/account/MyAccount.vue'
import MyPermissions from '../resources/vue/account/MyPermissions.vue'
import MyApiKeys from '../resources/vue/account/MyApiKeys.vue'

import { fetchAppStatus } from './appStatus.js'
import { hasPermission, canAccessIamFromStatus, canAccessControlPanelFromStatus } from './rbacAccess.js'

import {
	Notification03Icon,
	CodeSquareIcon,
	Settings01Icon,
	Configuration01Icon,
	File01Icon,
	DashboardSquare02Icon,
} from '@hugeicons/core-free-icons';

const controlPanelCrumb = { name: 'Control Panel', href: '/control-panel' }
const iamCrumb = { name: 'IAM', href: '/control-panel/iam' }
const accountCrumb = { name: 'My Account', href: '/account' }


const routes = [
  {
    path: '/',
    name: 'welcome',
    component: Welcome,
    meta: {
      title: 'Welcome',
      icon: Notification03Icon,
      breadcrumbs: () => [
        { name: 'Welcome' }
      ]
    }
  },
  {
    path: '/build-configs',
    name: 'buildConfigList',
    component: BuildConfigList,
    meta: {
      title: 'Build Configs',
      icon: Configuration01Icon,
      description: 'View and manage YAML build configurations',
      breadcrumbs: () => [
        { name: 'Build Configs' }
      ]
    }
  },
  {
    path: '/build-config/:name',
    name: 'buildConfig',
    component: BuildConfigView,
    props: true,
    meta: {
      title: 'View Build Config',
      breadcrumbs: () => [
        { name: 'Build Configs', href: '/build-configs' },
        { name: 'View Build Config' }
      ]
    }
  },
  {
    path: '/templates',
    name: 'templateList',
    component: TemplateList,
    meta: {
      title: 'Templates',
      icon: CodeSquareIcon,
      description: 'Browse built-in and custom site templates',
      breadcrumbs: () => [
        { name: 'Templates' }
      ]
    }
  },
  {
    path: '/template/add',
    name: 'templateAdd',
    component: TemplateAdd,
    meta: {
      breadcrumbs: () => [
        { name: 'Templates', href: '/templates' },
        { name: 'Add Template' }
      ]
    }
  },
  {
    path: '/template/:name',
    name: 'templateView',
    component: TemplateView,
    props: true,
    meta: {
      breadcrumbs: () => [
        { name: 'Templates', href: '/templates' },
        { name: 'View Template' }
      ]
    }
  },
  {
    path: '/system',
    name: 'systemDetails',
    component: SystemDetails,
    meta: {
      title: 'System Details',
      icon: Settings01Icon,
      description: 'Paths, version info, and system status',
      breadcrumbs: () => [
        { name: 'System Details' }
      ]
    }
  },
  {
    path: '/data-files',
    name: 'dataFileList',
    component: DataFileList,
    meta: {
      title: 'Data Files',
      icon: File01Icon,
      description: 'View data files used by your builds',
      breadcrumbs: () => [
        { name: 'Data Files' }
      ]
    }
  },
  {
    path: '/data-file/:buildConfigName/:datafileName',
    name: 'dataFileView',
    component: DataFileView,
    props: true,
    meta: {
      breadcrumbs: () => [
        { name: 'Data Files', href: '/data-files' },
        { name: 'View Data File' }
      ]
    }
  },
  {
    path: '/control-panel',
    name: 'controlPanel',
    component: ControlPanel,
    meta: {
      title: 'Control Panel',
      icon: DashboardSquare02Icon,
      description: 'System administration: users, groups, roles and permissions',
      requiresControlPanel: true,
      breadcrumbs: () => [{ name: 'Control Panel' }]
    }
  },
  {
    path: '/control-panel/iam',
    name: 'iamHub',
    component: IamHub,
    meta: {
      title: 'Identity & Access Management',
      requiresIam: true,
      breadcrumbs: () => [controlPanelCrumb, { name: 'IAM' }]
    }
  },
  {
    path: '/control-panel/iam/users',
    name: 'iamUsers',
    component: IamUsers,
    meta: {
      title: 'Users',
      permission: 'users.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'Users' }]
    }
  },
  {
    path: '/control-panel/iam/users/:id',
    name: 'iamUserDetails',
    component: IamUserDetails,
    props: true,
    meta: {
      title: 'User',
      permission: 'users.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'Users', href: '/control-panel/iam/users' }, { name: 'User' }]
    }
  },
  {
    path: '/control-panel/iam/groups',
    name: 'iamGroups',
    component: IamUserGroups,
    meta: {
      title: 'User Groups',
      permission: 'usergroups.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'User Groups' }]
    }
  },
  {
    path: '/control-panel/iam/groups/:id',
    name: 'iamGroupDetails',
    component: IamUserGroupDetails,
    props: true,
    meta: {
      title: 'User Group',
      permission: 'usergroups.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'User Groups', href: '/control-panel/iam/groups' }, { name: 'User Group' }]
    }
  },
  {
    path: '/control-panel/iam/roles',
    name: 'iamRoles',
    component: IamRoles,
    meta: {
      title: 'Roles',
      permission: 'rbac.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'Roles' }]
    }
  },
  {
    path: '/control-panel/iam/roles/:id',
    name: 'iamRoleDetails',
    component: IamRoleDetails,
    props: true,
    meta: {
      title: 'Role',
      permission: 'rbac.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'Roles', href: '/control-panel/iam/roles' }, { name: 'Role' }]
    }
  },
  {
    path: '/control-panel/iam/permissions',
    name: 'iamPermissions',
    component: IamPermissions,
    meta: {
      title: 'Permission Catalog',
      permission: 'rbac.view',
      breadcrumbs: () => [controlPanelCrumb, iamCrumb, { name: 'Permission Catalog' }]
    }
  },
  {
    path: '/account',
    name: 'myAccount',
    component: MyAccount,
    meta: {
      title: 'My Account',
      breadcrumbs: () => [{ name: 'My Account' }]
    }
  },
  {
    path: '/account/permissions',
    name: 'myPermissions',
    component: MyPermissions,
    meta: {
      title: 'My Permissions',
      breadcrumbs: () => [accountCrumb, { name: 'My Permissions' }]
    }
  },
  {
    path: '/account/api-keys',
    name: 'myApiKeys',
    component: MyApiKeys,
    meta: {
      title: 'API Keys',
      breadcrumbs: () => [accountCrumb, { name: 'API Keys' }]
    }
  }
]

const router = createRouter({
  history: createWebHistory('/webui'),
  routes
})

// routeAllowed mirrors the server-side permission checks so users are not shown pages whose
// RPCs would be refused. It is not a security boundary.
export function routeAllowed(route, st) {
  const meta = route.meta || {}
  if (meta.requiresControlPanel && !canAccessControlPanelFromStatus(st)) {
    return false
  }
  if (meta.requiresIam && !canAccessIamFromStatus(st)) {
    return false
  }
  if (meta.permission && !hasPermission(st, meta.permission)) {
    return false
  }
  return true
}

router.beforeEach(async (to) => {
  let st
  try {
    st = await fetchAppStatus()
  } catch {
    return true // App.vue shows the connection error
  }
  // Logged-out users see the login form in App.vue whatever the route; it is checked again after login.
  if (!st?.isLoggedIn) {
    return true
  }
  return routeAllowed(to, st) ? true : { name: 'welcome' }
})

export default router
