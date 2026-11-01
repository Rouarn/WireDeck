<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { t } from '../i18n'
import starUrl from '../assets/splash/star-four.svg'
import starfieldUrl from '../assets/splash/starfield.svg'

const leaving = ref(false)
const mounted = ref(true)

let leaveTimer = 0
let removeTimer = 0

onMounted(() => {
  // 至少展示 1.5s，再淡出 0.35s 后卸载。
  leaveTimer = window.setTimeout(() => {
    leaving.value = true
    removeTimer = window.setTimeout(() => {
      mounted.value = false
    }, 360)
  }, 1500)
})

onBeforeUnmount(() => {
  window.clearTimeout(leaveTimer)
  window.clearTimeout(removeTimer)
})
</script>

<template>
  <Transition name="splash-fade">
    <div v-if="mounted" class="splash" :class="{ leaving }" aria-hidden="true">
      <img class="starfield" :src="starfieldUrl" alt="" />
      <div class="glow"></div>

      <div class="hero">
        <div class="star-wrap">
          <span class="ring"></span>
          <img class="star" :src="starUrl" alt="WireDeck" />
        </div>
        <h1 class="brand">WireDeck</h1>
        <p class="tagline">{{ t('app.tagline') }}</p>
      </div>

      <div class="bar"><i></i></div>
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
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background:
    radial-gradient(1200px 700px at 50% 38%, rgba(120, 70, 220, 0.28), transparent 65%),
    radial-gradient(900px 520px at 50% 105%, rgba(59, 130, 246, 0.18), transparent 70%),
    linear-gradient(180deg, #0b0816 0%, #120b24 48%, #0a0714 100%);
  opacity: 1;
  transition: opacity 0.36s ease;
}

.splash.leaving {
  opacity: 0;
}

.starfield {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.5;
  animation: field-in 1.2s ease both;
}

.glow {
  position: absolute;
  width: 520px;
  height: 520px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(162, 127, 194, 0.35), rgba(95, 190, 220, 0.12) 45%, transparent 70%);
  filter: blur(8px);
  animation: glow-pulse 3.2s ease-in-out infinite;
}

.hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  transform: translateY(-6px);
}

.star-wrap {
  position: relative;
  display: grid;
  place-items: center;
  width: 168px;
  height: 168px;
  margin-bottom: 10px;
}

.ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 1px solid rgba(194, 176, 255, 0.22);
  box-shadow:
    0 0 0 1px rgba(194, 176, 255, 0.08) inset,
    0 0 60px rgba(130, 90, 220, 0.25);
  animation: ring-breathe 3s ease-in-out infinite;
}

.star {
  width: 108px;
  height: 108px;
  filter: drop-shadow(0 10px 34px rgba(0, 0, 0, 0.45));
  animation:
    star-in 0.85s cubic-bezier(0.2, 0.8, 0.2, 1) both,
    star-float 5.5s ease-in-out 0.9s infinite;
}

.brand {
  margin: 0;
  color: #ffffff;
  font-family: 'Archivo Splash', 'PingFang SC', 'Helvetica Neue', sans-serif;
  font-size: 40px;
  font-weight: 600;
  letter-spacing: 0.4px;
  line-height: 1.1;
  animation: text-in 0.65s ease 0.18s both;
}

.tagline {
  margin: 0;
  color: rgba(222, 214, 255, 0.66);
  font-size: 13px;
  letter-spacing: 3px;
  animation: text-in 0.65s ease 0.3s both;
}

.bar {
  position: absolute;
  bottom: 64px;
  width: 132px;
  height: 2px;
  overflow: hidden;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.1);
  animation: text-in 0.5s ease 0.35s both;
}

.bar > i {
  display: block;
  width: 100%;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, rgba(240, 153, 95, 0.9), rgba(95, 190, 220, 0.9), rgba(162, 127, 194, 0.9));
  transform-origin: left center;
  animation: bar-fill 1.45s cubic-bezier(0.25, 0.8, 0.3, 1) both;
}

@keyframes field-in {
  from { opacity: 0; }
  to { opacity: 0.5; }
}

@keyframes star-in {
  from {
    opacity: 0;
    transform: scale(0.78) rotate(-14deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(0deg);
  }
}

@keyframes star-float {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-5px) rotate(2.5deg); }
}

@keyframes ring-breathe {
  0%, 100% { opacity: 0.55; transform: scale(1); }
  50% { opacity: 0.95; transform: scale(1.04); }
}

@keyframes glow-pulse {
  0%, 100% { opacity: 0.55; transform: scale(0.96); }
  50% { opacity: 0.9; transform: scale(1.06); }
}

@keyframes text-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes bar-fill {
  from { transform: scaleX(0); }
  to { transform: scaleX(1); }
}

.splash-fade-leave-active {
  transition: opacity 0.36s ease;
}

.splash-fade-enter-active {
  transition: none;
}

@media (prefers-reduced-motion: reduce) {
  .starfield,
  .glow,
  .ring,
  .star,
  .brand,
  .tagline,
  .bar,
  .bar > i {
    animation: none !important;
  }
}
</style>
