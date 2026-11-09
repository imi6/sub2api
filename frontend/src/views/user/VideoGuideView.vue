<template>
  <AppLayout>
    <div data-testid="video-guide-shell" class="mx-auto max-w-7xl space-y-6">
      <header class="card overflow-hidden border-primary-200 bg-gradient-to-br from-primary-50 to-white p-5 dark:border-primary-800/50 dark:from-primary-950/40 dark:to-dark-900 sm:p-8">
        <p class="text-xs font-semibold uppercase tracking-widest text-primary-600 dark:text-primary-400">{{ t('videoGuide.eyebrow') }}</p>
        <h2 class="mt-3 text-2xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-3xl">{{ t('videoGuide.title') }}</h2>
        <p class="mt-3 max-w-3xl text-sm leading-7 text-gray-600 dark:text-gray-300">{{ t('videoGuide.intro') }}</p>

        <dl class="mt-5 grid gap-3 text-sm md:grid-cols-2">
          <div class="min-w-0 rounded-xl border border-primary-100 bg-white/80 p-4 dark:border-dark-700 dark:bg-dark-800/80">
            <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('videoGuide.baseUrl') }}</dt>
            <dd class="mt-2 break-all font-mono text-gray-900 dark:text-white">{{ baseUrl }}</dd>
          </div>
          <div class="min-w-0 rounded-xl border border-primary-100 bg-white/80 p-4 dark:border-dark-700 dark:bg-dark-800/80">
            <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('videoGuide.auth') }}</dt>
            <dd class="mt-2 break-all font-mono text-gray-900 dark:text-white">Authorization: Bearer YOUR_API_KEY</dd>
          </div>
        </dl>
        <p class="mt-3 text-xs leading-6 text-gray-500 dark:text-gray-400">{{ t('videoGuide.baseUrlHint') }}</p>
        <div class="mt-4 flex flex-wrap gap-3 text-sm font-medium">
          <router-link to="/keys" class="inline-flex items-center gap-1.5 text-primary-600 hover:underline dark:text-primary-400">
            <Icon name="key" size="sm" />{{ t('videoGuide.apiKeyLink') }}
          </router-link>
          <router-link to="/available-channels" class="inline-flex items-center gap-1.5 text-primary-600 hover:underline dark:text-primary-400">
            <Icon name="grid" size="sm" />{{ t('videoGuide.modelsLink') }}
          </router-link>
        </div>
        <ol class="mt-6 grid gap-2 sm:grid-cols-2 2xl:grid-cols-4">
          <li v-for="(step, index) in flow" :key="step" class="flex items-center gap-3 rounded-xl bg-white/70 px-3 py-3 text-sm text-gray-700 dark:bg-dark-800/70 dark:text-gray-200">
            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary-100 text-xs font-semibold text-primary-700 dark:bg-primary-900/50 dark:text-primary-300">{{ index + 1 }}</span>
            {{ t(`videoGuide.flow.${step}`) }}
          </li>
        </ol>
      </header>

      <details class="card p-4 xl:hidden">
        <summary class="cursor-pointer text-sm font-semibold text-gray-900 dark:text-white">{{ t('videoGuide.navTitle') }}</summary>
        <nav :aria-label="t('videoGuide.navTitle')" class="mt-3 grid gap-1 sm:grid-cols-2">
          <router-link v-for="(section, index) in sections" :key="section.id" :to="{ hash: `#${section.id}` }" class="rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-primary-50 hover:text-primary-700 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-primary-300">
            <span class="mr-2 font-mono text-xs text-gray-400">{{ String(index + 1).padStart(2, '0') }}</span>{{ section.title }}
          </router-link>
        </nav>
      </details>

      <div class="grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_210px]">
        <article class="min-w-0 space-y-6">
          <section v-for="(section, sectionIndex) in sections" :id="section.id" :key="section.id" :aria-labelledby="`${section.id}-title`" class="card scroll-mt-24 p-5 sm:p-6">
            <div class="flex items-baseline gap-3 border-b border-gray-100 pb-4 dark:border-dark-700">
              <span class="font-mono text-sm font-semibold text-primary-600 dark:text-primary-400">{{ String(sectionIndex + 1).padStart(2, '0') }}</span>
              <h2 :id="`${section.id}-title`" class="text-lg font-semibold text-gray-900 dark:text-white">{{ section.title }}</h2>
            </div>
            <p class="mt-4 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ section.intro }}</p>

            <template v-for="(block, blockIndex) in section.blocks" :key="blockIndex">
              <div v-if="block.type === 'note'" class="mt-4 rounded-lg border-l-4 px-4 py-3 text-sm leading-7" :class="block.warning ? 'border-amber-400 bg-amber-50 text-amber-900 dark:border-amber-600 dark:bg-amber-900/20 dark:text-amber-200' : 'border-primary-400 bg-primary-50 text-primary-900 dark:border-primary-600 dark:bg-primary-900/20 dark:text-primary-200'">
                {{ block.text }}
              </div>

              <ul v-else-if="block.type === 'list'" class="mt-4 list-disc space-y-2 pl-5 text-sm leading-7 text-gray-600 dark:text-gray-300">
                <li v-for="item in block.items" :key="item">{{ item }}</li>
              </ul>

              <div v-else-if="block.type === 'code'" class="mt-5 min-w-0 overflow-hidden rounded-xl border border-gray-800 bg-gray-900">
                <div class="flex items-center justify-between gap-3 border-b border-gray-700 bg-gray-800 px-4 py-2.5">
                  <h3 class="min-w-0 text-xs font-medium leading-5 text-gray-200">{{ block.label }}</h3>
                  <button type="button" :aria-label="`${t('videoGuide.copy')} ${block.label}`" class="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-gray-600 px-2.5 py-1 text-xs text-gray-200 transition-colors hover:border-gray-400 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-400" @click="copyExample(block.key)">
                    <Icon :name="copied && copiedExample === block.key ? 'check' : 'copy'" size="xs" />
                    {{ copied && copiedExample === block.key ? t('videoGuide.copied') : t('videoGuide.copy') }}
                  </button>
                </div>
                <pre class="overflow-x-auto p-4 text-xs leading-6 text-gray-100 sm:p-5"><code>{{ examples[block.key] }}</code></pre>
              </div>

              <div v-else-if="block.type === 'table'" class="mt-4 overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700">
                <table class="w-full min-w-[600px] text-left text-sm">
                  <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                    <tr>
                      <th v-for="column in block.columns" :key="column.key" scope="col" class="px-4 py-3 font-medium">{{ column.label }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 text-gray-700 dark:divide-dark-700 dark:text-gray-200">
                    <tr v-for="(row, rowIndex) in block.rows" :key="rowIndex">
                      <td v-for="column in block.columns" :key="column.key" class="px-4 py-3 align-top leading-6" :class="column.code ? 'whitespace-nowrap font-mono text-xs' : 'min-w-[130px]'">
                        <span v-if="column.key === 'method'" class="rounded px-2 py-1 font-semibold" :class="row[column.key] === 'POST' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'">{{ row[column.key] }}</span>
                        <template v-else>{{ row[column.key] }}</template>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </template>
          </section>
        </article>

        <aside class="hidden xl:block">
          <nav :aria-label="t('videoGuide.navTitle')" class="sticky top-24 max-h-[calc(100vh-7rem)] overflow-y-auto rounded-xl border border-gray-200 bg-white/80 p-3 dark:border-dark-700 dark:bg-dark-900/80">
            <p class="px-3 pb-2 pt-1 text-xs font-semibold text-gray-500 dark:text-gray-400">{{ t('videoGuide.navTitle') }}</p>
            <router-link v-for="(section, index) in sections" :key="section.id" :to="{ hash: `#${section.id}` }" :aria-current="activeSection === section.id ? 'location' : undefined" class="flex gap-2 rounded-lg px-3 py-2 text-sm transition-colors" :class="activeSection === section.id ? 'bg-primary-50 font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-white'">
              <span class="pt-0.5 font-mono text-xs opacity-60">{{ String(index + 1).padStart(2, '0') }}</span>{{ section.title }}
            </router-link>
          </nav>
        </aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { buildVideoGuideExamples, type VideoGuideExampleKey } from './videoGuideExamples'

interface GuideColumn {
  key: string
  label: string
  code?: boolean
}

type GuideBlock =
  | { type: 'note'; text: string; warning?: boolean }
  | { type: 'list'; items: string[] }
  | { type: 'code'; key: VideoGuideExampleKey; label: string }
  | { type: 'table'; columns: GuideColumn[]; rows: Record<string, string>[] }

interface GuideSection {
  id: string
  title: string
  intro: string
  blocks: GuideBlock[]
}

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const { copyToClipboard, copied } = useClipboard()
const copiedExample = ref<VideoGuideExampleKey | null>(null)
const flow = ['prepare', 'create', 'poll', 'download']

// 兼容后台配置根地址或带 /v1 的地址，确保示例不会重复拼接 /v1。
const baseUrl = computed(() => (appStore.apiBaseUrl.trim() || window.location.origin)
  .replace(/\/+$/, '').replace(/\/v1$/i, ''))
const examples = computed(() => buildVideoGuideExamples(baseUrl.value))
const activeSection = computed(() => route.hash.slice(1) || 'overview')

function code(key: VideoGuideExampleKey): GuideBlock {
  return { type: 'code', key, label: t(`videoGuide.examples.${key}`) }
}

function note(section: string, key: string, warning = false): GuideBlock {
  return { type: 'note', text: t(`videoGuide.sections.${section}.notes.${key}`), warning }
}

function list(section: string, keys: string[]): GuideBlock {
  return { type: 'list', items: keys.map(key => t(`videoGuide.sections.${section}.notes.${key}`)) }
}

function column(key: string, isCode = false): GuideColumn {
  return { key, label: t(`videoGuide.columns.${key}`), code: isCode }
}

function section(id: string, blocks: GuideBlock[]): GuideSection {
  return { id, title: t(`videoGuide.sections.${id}.title`), intro: t(`videoGuide.sections.${id}.intro`), blocks }
}

const sections = computed<GuideSection[]>(() => [
  section('overview', [
    list('overview', ['auth', 'models', 'async']),
    code('models'),
    { type: 'note', text: t('videoGuide.placeholderHint') },
  ]),
  section('endpoints', [
    {
      type: 'table', columns: [column('method', true), column('path', true), column('purpose')],
      rows: [
        { method: 'GET', path: '/v1/models', purpose: t('videoGuide.endpointPurpose.models') },
        { method: 'POST', path: '/v1/videos', purpose: t('videoGuide.endpointPurpose.create') },
        { method: 'GET', path: '/v1/videos/{id}', purpose: t('videoGuide.endpointPurpose.status') },
        { method: 'GET', path: '/v1/videos/{id}/content', purpose: t('videoGuide.endpointPurpose.content') },
        { method: 'POST', path: '/v1/videos/generations', purpose: t('videoGuide.endpointPurpose.createAlias') },
      ],
    },
    list('endpoints', ['aliases', 'ownership']),
  ]),
  section('parameters', [
    {
      type: 'table', columns: [column('field', true), column('type', true), column('required'), column('description')],
      rows: [
        ['model', 'model', 'string', 'yes'],
        ['prompt', 'prompt', 'string', 'conditional'],
        ['duration', 'duration', 'number', 'no'],
        ['ratio', 'ratio', 'string', 'no'],
        ['resolution', 'resolution', 'string', 'no'],
        ['firstFrame', 'first_frame', 'string | object', 'no'],
        ['lastFrame', 'last_frame', 'string | object', 'no'],
        ['referenceImages', 'reference_images', 'array', 'no'],
        ['referenceVideos', 'reference_videos', 'array', 'no'],
        ['referenceAudios', 'reference_audios', 'array', 'no'],
        ['seconds', 'seconds', 'number | string', 'no'],
        ['size', 'size', 'string', 'no'],
      ].map(([key, field, type, required]) => ({
        field, type, required: t(`videoGuide.requirements.${required}`), description: t(`videoGuide.params.${key}`),
      })),
    },
    list('parameters', ['aliases', 'conditional', 'limits']),
    { type: 'note', text: t('videoGuide.params.other') },
  ]),
  section('create', [
    code('create'), code('createResponse'),
    list('create', ['saveId', 'response']),
    note('create', 'retry', true),
  ]),
  section('media', [
    note('media', 'urls'),
    code('firstLast'), note('media', 'firstLast'),
    code('references'), note('media', 'references'),
    code('multimodal'), note('media', 'multimodal'),
    note('media', 'upload', true),
  ]),
  section('models', [
    {
      type: 'table', columns: [column('mode'), column('limits')],
      rows: ['text', 'frames', 'references', 'multimodal'].map(key => ({
        mode: t(`videoGuide.modelModes.${key}`), limits: t(`videoGuide.modelLimits.${key}`),
      })),
    },
    list('models', ['catalog', 'capabilities']),
    note('models', 'billing', true),
  ]),
  section('polling', [
    code('polling'), code('pollingResponse'),
    {
      type: 'table', columns: [column('status', true), column('action')],
      rows: [
        ['queued / pending', 'queued'],
        ['processing / in_progress / running', 'processing'],
        ['completed / succeeded / success / done', 'completed'],
        ['failed / error / cancelled / canceled / expired', 'failed'],
      ].map(([status, key]) => ({ status, action: t(`videoGuide.statuses.${key}`) })),
    },
    { type: 'note', text: t('videoGuide.statuses.unknown') },
    {
      type: 'table', columns: [column('field', true), column('description')],
      rows: [
        ['id / task_id / request_id', 'id'], ['status / state', 'status'], ['progress', 'progress'],
        ['url / video.url / content.video_url / output.url', 'result'], ['error / fail_reason', 'error'],
      ].map(([field, key]) => ({ field, description: t(`videoGuide.responses.${key}`) })),
    },
    list('polling', ['interval', 'statuses', 'timeout']),
  ]),
  section('download', [
    code('download'),
    list('download', ['content', 'url', 'range']),
  ]),
  section('python', [
    list('python', ['setup', 'behavior']),
    code('python'),
  ]),
  section('errors', [
    code('error'),
    {
      type: 'table', columns: [column('http', true), column('action')],
      rows: [
        ['400', 'badRequest'], ['401', 'unauthorized'], ['403', 'forbidden'], ['404', 'notFound'],
        ['409', 'conflict'], ['413', 'tooLarge'], ['415', 'unsupported'], ['429', 'rateLimit'],
        ['500 / 502 / 503 / 504', 'upstream'],
      ].map(([http, key]) => ({ http, action: t(`videoGuide.errors.${key}`) })),
    },
    list('errors', ['body', 'trace']),
    note('errors', 'retry', true),
  ]),
  section('compatibility', [list('compatibility', ['protocol', 'fields', 'records'])]),
])

async function copyExample(key: VideoGuideExampleKey) {
  if (await copyToClipboard(examples.value[key], t('videoGuide.copied'))) {
    copiedExample.value = key
  }
}

onMounted(async () => {
  // 首次导航在应用挂载前完成，补齐直接打开章节链接时的定位。
  await nextTick()
  if (route.hash && window.scrollY === 0) {
    const target = document.getElementById(route.hash.slice(1))
    if (target) window.scrollTo({ top: window.scrollY + target.getBoundingClientRect().top - 88 })
  }
  void appStore.fetchPublicSettings().catch((error) => {
    console.error('Failed to load settings:', error)
  })
})
</script>
