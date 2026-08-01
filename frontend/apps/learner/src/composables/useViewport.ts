// 移动端适配工具：设备检测、安全区域、Viewport 元信息
import { computed, onMounted, onBeforeUnmount, ref } from 'vue';

const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1200);
const viewportHeight = ref(typeof window !== 'undefined' ? window.innerHeight : 800);
const dpr = ref(typeof window !== 'undefined' ? window.devicePixelRatio : 1);

const MOBILE_BREAKPOINT = 768;
const TABLET_BREAKPOINT = 1024;

export function useViewport() {
  const isMobile = computed(() => viewportWidth.value <= MOBILE_BREAKPOINT);
  const isTablet = computed(
    () => viewportWidth.value > MOBILE_BREAKPOINT && viewportWidth.value <= TABLET_BREAKPOINT,
  );
  const isDesktop = computed(() => viewportWidth.value > TABLET_BREAKPOINT);
  const isTouch = computed(
    () =>
      typeof window !== 'undefined' &&
      ('ontouchstart' in window || navigator.maxTouchPoints > 0),
  );

  const safeAreaInsetTop = computed(() => {
    if (typeof getComputedStyle !== 'function') return 0;
    const val = getComputedStyle(document.documentElement).getPropertyValue('--sat').trim();
    return parseInt(val, 10) || 0;
  });
  const safeAreaInsetBottom = computed(() => {
    if (typeof getComputedStyle !== 'function') return 0;
    const val = getComputedStyle(document.documentElement).getPropertyValue('--sab').trim();
    return parseInt(val, 10) || 0;
  });

  function handleResize() {
    viewportWidth.value = window.innerWidth;
    viewportHeight.value = window.innerHeight;
    dpr.value = window.devicePixelRatio || 1;
  }

  onMounted(() => {
    window.addEventListener('resize', handleResize);
    window.addEventListener('orientationchange', handleResize);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('resize', handleResize);
    window.removeEventListener('orientationchange', handleResize);
  });

  return {
    viewportWidth,
    viewportHeight,
    dpr,
    isMobile,
    isTablet,
    isDesktop,
    isTouch,
    safeAreaInsetTop,
    safeAreaInsetBottom,
  };
}

/** 判断是否为 iOS Safari 类浏览器（需注意安全区域） */
export function isIOSSafari(): boolean {
  if (typeof navigator === 'undefined') return false;
  return /iPhone|iPad|iPod/i.test(navigator.userAgent) && /WebKit/i.test(navigator.userAgent);
}
