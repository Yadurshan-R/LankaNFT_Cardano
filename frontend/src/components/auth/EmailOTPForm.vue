<template>
  <div class="otp-form">
    <!-- Step 1: Enter email -->
    <div v-if="step === 'email'">
      <h2 class="form-title">Sign in with Email</h2>
      <p class="form-subtitle">We'll send a 6-digit code to your inbox</p>

      <div class="form-body">
        <BaseInput
          v-model="email"
          label="Email address"
          type="email"
          placeholder="you@example.com"
          :error="auth.error ?? undefined"
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
        />

        <BaseButton
          variant="primary"
          :loading="auth.isLoading"
          @click="handleVerifyOTP"
        >
          Verify & Sign In
        </BaseButton>

        <!-- Resend OTP -->
        <button class="resend-btn" :disabled="auth.isLoading" @click="step = 'email'">
          ← Use a different email
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const auth = useAuthStore()
const router = useRouter()

// Two steps: 'email' → 'otp'
const step = ref<'email' | 'otp'>('email')
const email = ref('')
const code = ref('')

// Step 1 — request OTP
async function handleRequestOTP() {
  if (!email.value) return
  const success = await auth.requestOTP(email.value)
  if (success) step.value = 'otp'
}

// Step 2 — verify OTP and redirect to dashboard
async function handleVerifyOTP() {
  if (!code.value || code.value.length !== 6) return
  const success = await auth.verifyOTP(email.value, code.value)
  if (success) router.push('/')
}
</script>

<style scoped>
.otp-form {
  width: 100%;
}
.form-title {
  font-size: 22px;
  font-weight: 600;
  color: #111;
  margin-bottom: 6px;
}
.form-subtitle {
  font-size: 14px;
  color: #666;
  margin-bottom: 24px;
}
.form-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.resend-btn {
  background: none;
  border: none;
  font-size: 13px;
  color: #534ab7;
  cursor: pointer;
  text-align: left;
  padding: 0;
}
.resend-btn:hover {
  text-decoration: underline;
}
</style>