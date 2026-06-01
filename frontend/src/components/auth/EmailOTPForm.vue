<template>
  <div class="otp-form">

    <!-- Banners -->
    <div v-if="reason === 'session_expired'" class="banner banner--amber">
      <Clock :size="14" /> Your session expired. Please sign in again.
    </div>
    <div v-else-if="reason === 'backend_down'" class="banner banner--red">
      <AlertCircle :size="14" /> Service temporarily unavailable. Try again shortly.
    </div>

    <!-- Step 1: Email -->
    <div v-if="step === 'email'" class="form-body">
      <BaseInput
        v-model="email"
        label="Email address"
        type="email"
        placeholder="you@example.com"
        :error="auth.error ?? undefined"
        @keyup.enter="handleRequestOTP"
      />
      <BaseButton variant="primary" :loading="auth.isLoading" @click="handleRequestOTP">
        Send OTP
      </BaseButton>
      <p class="form-note">
        We'll send a 6-digit code to your inbox.
      </p>
    </div>

    <!-- Step 2: OTP -->
    <div v-if="step === 'otp'" class="form-body">
      <p class="otp-sent">
        Code sent to <strong>{{ email }}</strong>
      </p>
      <BaseInput
        v-model="code"
        label="6-digit code"
        type="text"
        placeholder="000000"
        :error="auth.error ?? undefined"
        @keyup.enter="handleVerifyOTP"
      />
      <BaseButton variant="primary" :loading="auth.isLoading" @click="handleVerifyOTP">
        Verify &amp; Sign In
      </BaseButton>
      <div class="resend-row">
        <button class="link-btn" :disabled="resendCooldown > 0 || auth.isLoading" @click="handleResend">
          {{ resendCooldown > 0 ? `Resend in ${resendCooldown}s` : 'Resend code' }}
        </button>
        <span class="sep">·</span>
        <button class="link-btn" :disabled="auth.isLoading" @click="goBack">
          Change email
        </button>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircle, Clock } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import BaseInput  from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const auth   = useAuthStore()
const router = useRouter()
const route  = useRoute()

const reason = route.query.reason as string | undefined
const step   = ref<'email' | 'otp'>('email')
const email  = ref('')
const code   = ref('')

const resendCooldown = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

function startCountdown() {
  resendCooldown.value = 60
  timer = setInterval(() => {
    if (--resendCooldown.value <= 0) { clearInterval(timer!); timer = null }
  }, 1000)
}
onUnmounted(() => { if (timer) clearInterval(timer) })

async function handleRequestOTP() {
  if (!email.value) return
  const ok = await auth.requestOTP(email.value)
  if (ok) { step.value = 'otp'; startCountdown() }
}
async function handleResend() {
  if (resendCooldown.value > 0) return
  const ok = await auth.requestOTP(email.value)
  if (ok) startCountdown()
}
async function handleVerifyOTP() {
  if (!code.value || code.value.length !== 6) return
  const ok = await auth.verifyOTP(email.value, code.value)
  if (ok) router.push('/')
}
function goBack() {
  step.value = 'email'; code.value = ''
  resendCooldown.value = 0
  if (timer) { clearInterval(timer); timer = null }
}
</script>

<style scoped>
.otp-form { width: 100%; }

.banner {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 14px; border-radius: 10px;
  font-size: 13px; font-weight: 500; margin-bottom: 20px;
}
.banner--amber { background: #FAEEDA; color: #854F0B; }
.banner--red   { background: #FCEBEB; color: #791F1F; }

.form-body {
  display: flex; flex-direction: column; gap: 14px;
}

.form-note {
  font-size: 12px; color: #bbb; text-align: center; margin: 0;
}

.otp-sent {
  font-size: 13px; color: #666; margin: 0;
}
.otp-sent strong { color: #333; }

.resend-row {
  display: flex; align-items: center; gap: 8px;
}
.link-btn {
  background: none; border: none;
  font-size: 12px; color: #534AB7; font-weight: 500;
  cursor: pointer; padding: 0; transition: opacity 0.15s;
}
.link-btn:disabled { color: #bbb; cursor: not-allowed; }
.link-btn:not(:disabled):hover { text-decoration: underline; }
.sep { color: #ddd; font-size: 13px; }
</style>