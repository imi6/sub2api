<template>
  <AppLayout>
    <div data-testid="video-guide-shell" class="mx-auto max-w-[950px] space-y-6">
      <div class="card p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('videoGuide.overviewTitle') }}</h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('videoGuide.overviewDesc') }}</p>
        <ol class="mt-4 list-decimal space-y-1 pl-5 text-sm text-gray-700 dark:text-gray-200">
          <li>{{ t('videoGuide.flowCreate') }}</li>
          <li>{{ t('videoGuide.flowPoll') }}</li>
          <li>{{ t('videoGuide.flowDownload') }}</li>
        </ol>
        <dl class="mt-4 grid gap-3 text-sm sm:grid-cols-2">
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('videoGuide.baseUrl') }}</dt>
            <dd class="mt-1 break-all font-mono text-gray-900 dark:text-white">{{ baseUrl }}</dd>
          </div>
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('videoGuide.auth') }}</dt>
            <dd class="mt-1 break-all font-mono text-gray-900 dark:text-white">Authorization: Bearer sk-xxx</dd>
          </div>
        </dl>
        <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {{ t('videoGuide.keyHint') }}
          <router-link to="/keys" class="text-primary-600 hover:underline">{{ t('nav.apiKeys') }}</router-link>
          ·
          <router-link to="/available-channels" class="text-primary-600 hover:underline">{{ t('nav.availableChannels') }}</router-link>
        </p>
      </div>

      <div v-for="step in steps" :key="step.key" class="card p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t(`videoGuide.${step.key}.title`) }}</h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t(`videoGuide.${step.key}.desc`) }}</p>
        <p class="mt-3 font-mono text-sm text-gray-900 dark:text-white">
          <span class="mr-2 rounded bg-primary-100 px-1.5 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">{{ step.method }}</span>{{ step.path }}
        </p>
        <div class="relative mt-3">
          <button
            type="button"
            class="absolute right-2 top-2 rounded bg-gray-700 px-2 py-1 text-xs text-gray-100 hover:bg-gray-600"
            @click="copyToClipboard(step.code, t('videoGuide.copied'))"
          >
            {{ t('videoGuide.copy') }}
          </button>
          <pre class="overflow-x-auto rounded-lg bg-gray-900 p-4 pr-16 text-xs leading-relaxed text-gray-100"><code>{{ step.code }}</code></pre>
        </div>
        <ul v-if="step.notes" class="mt-3 list-disc space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-300">
          <li v-for="note in step.notes" :key="note">{{ t(`videoGuide.${step.key}.${note}`) }}</li>
        </ul>
      </div>

      <div class="card p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('videoGuide.params.title') }}</h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('videoGuide.params.desc') }}</p>
        <div class="mt-3 overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead class="text-gray-500 dark:text-gray-400">
              <tr>
                <th scope="col" class="py-2 pr-4 font-medium">{{ t('videoGuide.params.field') }}</th>
                <th scope="col" class="py-2 pr-4 font-medium">{{ t('videoGuide.params.required') }}</th>
                <th scope="col" class="py-2 font-medium">{{ t('videoGuide.params.meaning') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 text-gray-700 dark:divide-dark-700 dark:text-gray-200">
              <tr v-for="row in paramRows" :key="row.key">
                <td class="whitespace-nowrap py-2 pr-4 font-mono">{{ row.field }}</td>
                <td class="whitespace-nowrap py-2 pr-4">{{ row.required ? t('videoGuide.params.yes') : t('videoGuide.params.no') }}</td>
                <td class="py-2">{{ t(`videoGuide.params.rows.${row.key}`) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <h3 class="mt-6 font-semibold text-gray-900 dark:text-white">{{ t('videoGuide.params.imageExampleTitle') }}</h3>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('videoGuide.params.imageExampleDesc') }}</p>
        <div class="relative mt-3">
          <button
            type="button"
            class="absolute right-2 top-2 rounded bg-gray-700 px-2 py-1 text-xs text-gray-100 hover:bg-gray-600"
            @click="copyToClipboard(imageExampleCode, t('videoGuide.copied'))"
          >
            {{ t('videoGuide.copy') }}
          </button>
          <pre class="overflow-x-auto rounded-lg bg-gray-900 p-4 pr-16 text-xs leading-relaxed text-gray-100"><code>{{ imageExampleCode }}</code></pre>
        </div>
      </div>

      <div class="card p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('videoGuide.statuses.title') }}</h2>
        <div class="mt-3 overflow-x-auto">
          <table class="w-full text-left text-sm">
            <tbody class="divide-y divide-gray-100 text-gray-700 dark:divide-dark-700 dark:text-gray-200">
              <tr v-for="row in statusRows" :key="row.key">
                <td class="whitespace-nowrap py-2 pr-4 font-mono">{{ row.values }}</td>
                <td class="py-2">{{ t(`videoGuide.statuses.${row.key}`) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card border-amber-200 bg-amber-50 p-6 dark:border-amber-800/50 dark:bg-amber-900/20">
        <h2 class="text-lg font-semibold text-amber-900 dark:text-amber-200">{{ t('videoGuide.tips.title') }}</h2>
        <ul class="mt-3 list-disc space-y-1 pl-5 text-sm text-amber-900 dark:text-amber-100">
          <li v-for="tip in tips" :key="tip">{{ t(`videoGuide.tips.${tip}`) }}</li>
        </ul>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const exampleModel = '官转SD2.5-30图版-480P'

// 对外地址优先用后台配置的 API 地址，未配置时用当前站点地址。
const baseUrl = computed(() => (appStore.apiBaseUrl || window.location.origin).replace(/\/+$/, ''))

const steps = computed(() => [
  {
    key: 'create',
    method: 'POST',
    path: '/v1/videos',
    code: `curl -X POST "${baseUrl.value}/v1/videos" \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${exampleModel}",
    "prompt": "A cat walking slowly on a sunny beach, gentle waves in the background",
    "duration": 5,
    "ratio": "16:9",
    "resolution": "480p"
  }'

# Response
{"id":"vid_xxx","object":"video","model":"${exampleModel}","status":"queued"}`,
    notes: ['noteId', 'noteRetry'],
  },
  {
    key: 'status',
    method: 'GET',
    path: '/v1/videos/{id}',
    code: `curl "${baseUrl.value}/v1/videos/vid_xxx" \\
  -H "Authorization: Bearer sk-xxx"

# Response
{"id":"vid_xxx","status":"completed","duration":5,"resolution":"480p"}`,
    notes: ['notePoll'],
  },
  {
    key: 'download',
    method: 'GET',
    path: '/v1/videos/{id}/content',
    code: `curl -L "${baseUrl.value}/v1/videos/vid_xxx/content" \\
  -H "Authorization: Bearer sk-xxx" \\
  -o video.mp4`,
    notes: ['noteOnlyCompleted', 'noteRange'],
  },
])

const paramRows = [
  { key: 'model', field: 'model', required: true },
  { key: 'prompt', field: 'prompt', required: true },
  { key: 'duration', field: 'duration', required: false },
  { key: 'ratio', field: 'ratio', required: false },
  { key: 'resolution', field: 'resolution', required: false },
  { key: 'firstFrame', field: 'first_frame', required: false },
  { key: 'lastFrame', field: 'last_frame', required: false },
  { key: 'referenceImages', field: 'reference_images', required: false },
  { key: 'referenceVideos', field: 'reference_videos', required: false },
  { key: 'referenceAudios', field: 'reference_audios', required: false },
]

const imageExampleCode = computed(() => `curl -X POST "${baseUrl.value}/v1/videos" \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${exampleModel}",
    "prompt": "The girl in the picture turns around and smiles",
    "duration": 5,
    "ratio": "16:9",
    "reference_images": [
      { "image_url": "https://example.com/ref-1.png" },
      { "image_url": "https://example.com/ref-2.png" }
    ]
  }'`)

const statusRows = [
  { key: 'running', values: 'queued / processing' },
  { key: 'completed', values: 'completed' },
  { key: 'failed', values: 'failed' },
]

const tips = ['billing', 'keepId', 'limits', 'support']

onMounted(() => {
  void appStore.fetchPublicSettings().catch((error) => {
    console.error('Failed to load settings:', error)
  })
})
</script>
