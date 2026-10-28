<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import {
  SEEDANCE_PATH_FIELDS,
  SEEDANCE_PRESET_DEFAULTS,
  SEEDANCE_RESPONSE_FIELDS,
  validateSeedanceProfile,
  type SeedanceProfileForm,
  type SeedanceTextField
} from './seedanceProfile'

// Edits credentials.seedance_profile. Empty inputs fall back to the preset
// defaults (shown as placeholders), so only real overrides are stored.
const profile = defineModel<SeedanceProfileForm>({ required: true })
const { t } = useI18n()
const showAdvanced = ref(false)

const defaults = computed(() => SEEDANCE_PRESET_DEFAULTS[profile.value.preset])

const presetOptions = computed(() => [
  { value: 'ark', label: t('admin.accounts.seedance.presetArk') },
  { value: 'openai_videos', label: t('admin.accounts.seedance.presetOpenAIVideos') }
])

const billingOptions = computed(() => [
  {
    value: '',
    label: t('admin.accounts.seedance.billingDefault', {
      mode: t(`admin.accounts.seedance.billing.${defaults.value.billing_mode}`)
    })
  },
  { value: 'video', label: t('admin.accounts.seedance.billing.video') },
  { value: 'tokens', label: t('admin.accounts.seedance.billing.tokens') }
])

const requestFormatOptions = computed(() => [
  { value: '', label: t('admin.accounts.seedance.requestFormats.passthrough') },
  { value: 'sd25', label: t('admin.accounts.seedance.requestFormats.sd25') },
  { value: 'opaigc', label: t('admin.accounts.seedance.requestFormats.opaigc') }
])

const effectiveBillingMode = computed(() => profile.value.billing_mode || defaults.value.billing_mode)

// Usage tokens only matter for token billing; hide it otherwise to reduce noise.
const responseFields = computed(() =>
  SEEDANCE_RESPONSE_FIELDS.filter(
    (field) => field !== 'usage_tokens_field' || effectiveBillingMode.value === 'tokens'
  )
)

const validationError = computed(() => {
  const key = validateSeedanceProfile(profile.value)
  return key ? t(key) : ''
})

const placeholderFor = (field: SeedanceTextField) =>
  defaults.value[field] || t('admin.accounts.seedance.notSupported')

const resetOverrides = () => {
  for (const field of [...SEEDANCE_PATH_FIELDS, ...SEEDANCE_RESPONSE_FIELDS]) {
    profile.value[field] = ''
  }
  profile.value.billing_mode = ''
  profile.value.request_format = ''
}
</script>

<template>
  <div class="space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600" data-testid="seedance-profile-editor">
    <div>
      <label class="input-label">{{ t('admin.accounts.seedance.preset') }}</label>
      <Select v-model="profile.preset" :options="presetOptions" data-testid="seedance-profile-preset" />
      <p class="input-hint">
        {{ profile.preset === 'openai_videos'
          ? t('admin.accounts.seedance.presetOpenAIVideosDesc')
          : t('admin.accounts.seedance.presetArkDesc') }}
      </p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.seedance.requestFormat') }}</label>
      <Select v-model="profile.request_format" :options="requestFormatOptions" data-testid="seedance-profile-request-format" />
      <p class="input-hint">{{ t('admin.accounts.seedance.requestFormatHint') }}</p>
    </div>

    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <div v-for="field in SEEDANCE_PATH_FIELDS" :key="field">
        <label class="input-label" :for="`seedance-${field}`">{{ t(`admin.accounts.seedance.fields.${field}`) }}</label>
        <input
          :id="`seedance-${field}`"
          v-model="profile[field]"
          type="text"
          class="input font-mono text-xs"
          :placeholder="placeholderFor(field)"
          :data-testid="`seedance-profile-${field}`"
          autocomplete="off"
          spellcheck="false"
        />
      </div>
    </div>
    <p class="input-hint">{{ t('admin.accounts.seedance.pathsHint') }}</p>

    <div>
      <label class="input-label">{{ t('admin.accounts.seedance.billingMode') }}</label>
      <Select v-model="profile.billing_mode" :options="billingOptions" data-testid="seedance-profile-billing-mode" />
      <p class="input-hint">
        {{ effectiveBillingMode === 'video'
          ? t('admin.accounts.seedance.billingVideoHint')
          : t('admin.accounts.seedance.billingTokensHint') }}
      </p>
    </div>

    <button
      type="button"
      class="text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
      :aria-expanded="showAdvanced"
      data-testid="seedance-profile-advanced-toggle"
      @click="showAdvanced = !showAdvanced"
    >
      {{ showAdvanced ? t('admin.accounts.seedance.hideAdvanced') : t('admin.accounts.seedance.showAdvanced') }}
    </button>

    <div v-if="showAdvanced" class="space-y-3">
      <p class="input-hint">{{ t('admin.accounts.seedance.fieldsHint') }}</p>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div v-for="field in responseFields" :key="field">
          <label class="input-label" :for="`seedance-${field}`">{{ t(`admin.accounts.seedance.fields.${field}`) }}</label>
          <input
            :id="`seedance-${field}`"
            v-model="profile[field]"
            type="text"
            class="input font-mono text-xs"
            :placeholder="placeholderFor(field)"
            :data-testid="`seedance-profile-${field}`"
            autocomplete="off"
            spellcheck="false"
          />
        </div>
      </div>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        data-testid="seedance-profile-reset"
        @click="resetOverrides"
      >
        {{ t('admin.accounts.seedance.resetOverrides') }}
      </button>
    </div>

    <p
      v-if="validationError"
      role="alert"
      class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300"
      data-testid="seedance-profile-error"
    >
      {{ validationError }}
    </p>
  </div>
</template>
