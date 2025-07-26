import { createRouter, createWebHistory } from 'vue-router'

import Welcome from '../resources/vue/Welcome.vue'
import BuildConfigList from '../resources/vue/BuildConfigList.vue'
import BuildConfigView from '../resources/vue/BuildConfigView.vue'
import TemplateList from '../resources/vue/TemplateList.vue'
import TemplateView from '../resources/vue/TemplateView.vue'
import SystemDetails from '../resources/vue/SystemDetails.vue'

const routes = [
  {
    path: '/',
    name: 'welcome',
    component: Welcome,
    meta: {
      breadcrumb: [
        { name: 'Welcome' }
      ]
    }
  },
  {
    path: '/build-configs',
    name: 'buildConfigList',
    component: BuildConfigList,
    meta: {
      breadcrumb: [
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
      breadcrumb: [
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
      breadcrumb: [
        { name: 'Templates' }
      ]
    }
  },
  {
    path: '/template/:name',
    name: 'templateView',
    component: TemplateView,
    props: true,
    meta: {
      breadcrumb: [
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
      breadcrumb: [
        { name: 'System Details' }
      ]
    }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
