<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { t } from '../i18n'
import { useThemeStore } from '../stores/theme'
import starUrl from '../assets/splash/star-four.svg'
import iconUrl from '../assets/splash/app-icon.png'

const theme = useThemeStore()
const leaving = ref(false)
const mounted = ref(true)

const isDark = computed(() => theme.isDark)

let leaveTimer = 0
let removeTimer = 0

onMounted(() => {
  // 展示 1.55s，淡出 0.4s 后卸载。
  leaveTimer = window.setTimeout(() => {
    leaving.value = true
    removeTimer = window.setTimeout(() => {
      mounted.value = false
    }, 400)
  }, 1550)
})

onBeforeUnmount(() => {
  window.clearTimeout(leaveTimer)
  window.clearTimeout(removeTimer)
})
</script>

<template>
  <Transition name="splash-fade">
    <div v-if="mounted" class="splash" :class="{ dark: isDark, leaving }" aria-hidden="true">
      <img class="giant-star" :src="starUrl" alt="" />

      <div class="frame">
        <div class="meta meta-top">
          <span>WIREDECK / BRAND SYSTEM</span>
          <span>2026</span>
        </div>

        <div class="stage">
          <img class="app-icon" :src="iconUrl" alt="" />
          <div class="copy">
            <h1 class="brand">WireDeck</h1>
            <p class="tagline">{{ t('app.tagline') }}</p>
          </div>
        </div>

        <div class="meta meta-bottom">
          <div class="chips">
            <i class="chip orange"></i>
            <i class="chip blue"></i>
            <i class="chip purple"></i>
            <i class="chip yellow"></i>
          </div>
          <span>WIRELESS ADB · WOYUXIAJI.CN</span>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
@font-face {
  font-family: 'Archivo Splash';
  src: url('../assets/splash/Archivo.ttf') format('truetype');
  font-weight: 400 700;
  font-display: swap;
}

.splash {
  position: fixed;
  inset: 0;
  z-index: 4000;
  overflow: hidden;
  background-color: #f3f1ec;
  background-image:
    linear-gradient(to right, rgba(23, 21, 18, 0.055) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(23, 21, 18, 0.055) 1px, transparent 1px);
  background-size: 64px 64px;
  transition: opacity 0.4s ease;
}

.splash.dark {
  background-color: #0e0e0d;
  background-image:
    linear-gradient(to right, rgba(255, 255, 255, 0.05) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(255, 255, 255, 0.05) 1px, transparent 1px);
}

.splash.leaving {
  opacity: 0;
}

.giant-star {
  position: absolute;
  right: -180px;
  bottom: -300px;
  width: 880px;
  height: 880px;
  opacity: 0.1;
  transform: rotate(12deg);
  animation:
    star-drift 9s ease-in-out infinite,
    star-fade 0.9s ease both;
  pointer-events: none;
}

.splash.dark .giant-star {
  opacity: 0.16;
}

.frame {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 44px 56px;
  color: #171512;
}

.splash.dark .frame {
  color: #f4f1ea;
}

.meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-family: 'Archivo Splash', 'PingFang SC', sans-serif;
  font-size: 11px;
  letter-spacing: 0.18em;
  opacity: 0.42;
  animation: meta-in 0.7s ease 0.15s both;
}

.meta-bottom {
  align-items: flex-end;
}

.chips {
  display: flex;
  gap: 7px;
}

.chip {
  width: 11px;
  height: 11px;
  border-radius: 2px;
}

.chip.orange { background: #f0995f; }
.chip.blue { background: #5fbedc; }
.chip.purple { background: #a27fc2; }
.chip.yellow { background: #eddf55; }

.stage {
  display: flex;
  align-items: center;
  gap: 64px;
  margin-left: 4vw;
  margin-top: -4vh;
}

.app-icon {
  width: 208px;
  height: 208px;
  flex: 0 0 auto;
  filter: drop-shadow(0 18px 40px rgba(16, 12, 6, 0.18));
  animation: icon-in 0.8s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

.splash.dark .app-icon {
  filter: drop-shadow(0 18px 46px rgba(0, 0, 0, 0.55));
}

.copy {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.brand {
  margin: 0;
  font-family: 'Archivo Splash', 'PingFang SC', 'Helvetica Neue', sans-serif;
  font-size: clamp(64px, 7.2vw, 104px);
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
  animation: copy-in 0.75s cubic-bezier(0.2, 0.8, 0.2, 1) 0.08s both;
}

.tagline {
  margin: 0;
  font-size: 19px;
  letter-spacing: 0.08em;
  opacity: 0.58;
  animation: copy-in 0.75s cubic-bezier(0.2, 0.8, 0.2, 1) 0.18s both;
}

@keyframes icon-in {
  from {
    opacity: 0;
    transform: scale(0.92) translateY(8px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

@keyframes copy-in {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes meta-in {
  from { opacity: 0; }
  to { opacity: 0.42; }
}

@keyframes star-fade {
  from { opacity: 0; }
  to { opacity: 0.1; }
}

@keyframes star-drift {
  0%, 100% { transform: rotate(12deg) translate(0, 0); }
  50% { transform: rotate(14deg) translate(-10px, -8px); }
}

.splash-fade-leave-active {
  transition: opacity 0.4s ease;
}

@media (max-width: 900px) {
  .stage {
    gap: 32px;
    margin-left: 0;
  }

  .app-icon {
    width: 132px;
    height: 132px;
  }

  .tagline {
    font-size: 15px;
  }

  .giant-star {
    width: 560px;
    height: 560px;
    right: -140px;
    bottom: -180px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .giant-star,
  .app-icon,
  .brand,
  .tagline,
  .meta {
    animation: none !important;
  }
}
</style>
