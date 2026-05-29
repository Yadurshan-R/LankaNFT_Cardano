<template>
  <div class="otp-form">

    <!-- Session / backend status banners -->
    <div v-if="reason === 'session_expired'" class="banner banner--amber">
      <Clock :size="14" />
      Your session expired. Please sign in again.
    </div>
    <div v-else-if="reason === 'backend_down'" class="banner banner--red">
      <AlertCircle :size="14" />
      LankaNFT is temporarily unavailable. Please try again in a moment.
    </div>

    <!-- Step 1: Enter email -->
    <div v-if="step === 'email'">
      <h2 class="form-title">Sign in with Email</h2>
      <p class="form-subtitle">We'll send a 6-digit code to your inbox.</p>
      <div class="form-body">
        <BaseInput
          v-model="email"
          label="Email address"
          type="email"
          placeholder="you@example.com"
          :error="auth.error ?? undefined"
          @keyup.enter="handleRequestOTP"
        />
        <BaseButton
          variant="primary"
          :loading="auth.isLoading"
          @click="handleRequestOTP"
        >
          Send OTP
        </BaseButton>
      </div>
    </div>

    <!-- Step 2: Enter OTP code -->
    <div v-if="step === 'otp'">
      <h2 class="form-title">Check your email</h2>
      <p class="form-subtitle">
        Enter the 6-digit code sent to <strong>{{ email }}</strong>
      </p>
      <div class="form-body">
        <BaseInput
          v-model="code"
          label="OTP Code"
          type="text"
          placeholder="000000"
          :error="auth.error ?? undefined"
          @keyup.enter="handleVerifyOTP"
        />
        <BaseButton
          variant="primary"
          :loading="auth.isLoading"
          @click="handleVerifyOTP"
        >
          Verify & Sign In
        </BaseButton>

        <!-- Resend OTP with countdown timer -->
        <div class="resend-row">
          <button
            class="resend-btn"
            :disabled="resendCooldown > 0 || auth.isLoading"
            @click="handleResend"
          >
            <template v-if="resendCooldown > 0">
              Resend in {{ resendCooldown }}s
            </template>
            <template v-else>
              Resend OTP
            </template>
          </button>
          <span class="divider">·</span>
          <button class="resend-btn" :disabled="auth.isLoading" @click="goBack">
            ← Use a different email
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// EmailOTPForm.vue
//
// Two-step sign-in form: email → OTP verification.
//
// Improvements over v1:
//   - Reads ?reason query param to show session expired / backend down banners
//   - Resend OTP button with 60-second cooldown countdown
//     (prevents spam and tells the user exactly when they can resend)
//   - Timer clears on unmount to prevent memory leaks
// ─────────────────────────────────────────────────────────────────────────────

import { onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircle, Clock } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const auth   = useAuthStore()
const router = useRouter()
const route  = useRoute()

// Read reason from query param set by the 401/503 interceptor in api.ts
const reason = route.query.reason as string | undefined

// Two steps: 'email' → 'otp'
const step  = ref<'email' | 'otp'>('email')
const email = ref('')
const code  = ref('')

// ── Resend countdown ──────────────────────────────────────────────────────────
// Starts at 60 after OTP is sent. Counts down every second.
// Button is disabled and shows "Resend in Xs" while counting.
// Resets to 60 on each resend.
const resendCooldown = ref(0)
let countdownTimer: ReturnType<typeof setInterval> | null = null

function startCountdown() {
  resendCooldown.value = 60
  countdownTimer = setInterval(() => {
    resendCooldown.value--
    if (resendCooldown.value <= 0) {
      clearInterval(countdownTimer!)
      countdownTimer = null
    }
  }, 1000)
}

// Clear timer on unmount to prevent memory leaks
onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer)
})

// ── Handlers ──────────────────────────────────────────────────────────────────

// Step 1 — request OTP and start countdown
async function handleRequestOTP() {
  if (!email.value) return
  const success = await auth.requestOTP(email.value)
  if (success) {
    step.value = 'otp'
    startCountdown()
  }
}

// Resend OTP — restarts countdown
async function handleResend() {
  if (resendCooldown.value > 0) return
  const success = await auth.requestOTP(email.value)
  if (success) startCountdown()
}

// Step 2 — verify OTP and redirect to dashboard
async function handleVerifyOTP() {
  if (!code.value || code.value.length !== 6) return
  const success = await auth.verifyOTP(email.value, code.value)
  if (success) router.push('/')
}

// Go back to email step and clear countdown
function goBack() {
  step.value = 'email'
  code.value = ''
  resendCooldown.value = 0
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}
</script>

<style scoped>
.otp-form { width: 100%; }

/* Status banners */
.banner {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 14px; border-radius: 10px;
  font-size: 13px; font-weight: 500;
  margin-bottom: 20px;
}
.banner--amber { background: #FAEEDA; color: #854F0B; }
.banner--red   { background: #FCEBEB; color: #791F1F; }

.form-title    { font-size: 22px; font-weight: 600; color: #111; margin-bottom: 6px; }
.form-subtitle { font-size: 14px; color: #666; margin-bottom: 24px; }

.form-body {
  display: flex; flex-direction: column; gap: 16px;
}

/* Resend row */
.resend-row {
  display: flex; align-items: center; gap: 8px;
  flex-wrap: wrap;
}
.resend-btn {
  background: none; border: none;
  font-size: 13px; color: #534AB7;
  cursor: pointer; padding: 0;
  transition: opacity 0.15s;
}
.resend-btn:disabled {
  color: #aaa; cursor: not-allowed;
}
.resend-btn:not(:disabled):hover { text-decoration: underline; }
.divider { color: #ddd; font-size: 13px; }
</style>