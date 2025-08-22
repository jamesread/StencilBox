import { createRouter, createWebHistory } from 'vue-router'

import Welcome from '../resources/vue/Welcome.vue'
import BuildConfigList from '../resources/vue/BuildConfigList.vue'
import BuildConfigView from '../resources/vue/BuildConfigView.vue'
import TemplateList from '../resources/vue/TemplateList.vue'
import TemplateView from '../resources/vue/TemplateView.vue'
import TemplateAdd from '../resources/vue/TemplateAdd.vue'
import SystemDetails from '../resources/vue/SystemDetails.vue'

const routes = [
  {
    path: '/',
    name: 'welcome',
    component: Welcome,
    meta: {
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
      breadcrumbs: () => [
        { name: 'System Details' }
      ]
    }
  }
]

const router = createRouter({
  history: createWebHistory('/webui'),
  routes
})

export default router
