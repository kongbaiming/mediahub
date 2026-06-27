<template>
  <section
    v-if="row.type === 'hero-banner'"
    class="feed-hero"
    :class="{ 'feed-hero--immersive': immersive }"
    :style="heroBg"
  >
    <div class="feed-hero__overlay" />
    <div v-if="heroItem" class="feed-hero__content">
      <p v-if="row.title" class="feed-hero__eyebrow">{{ row.title }}</p>
      <h1 class="feed-hero__title">{{ heroItem.title }}</h1>
      <div class="feed-hero__meta">
        <span v-if="heroItem.year" class="feed-hero__chip">{{ heroItem.year }}</span>
        <span v-if="heroItem.rating" class="feed-hero__chip feed-hero__chip--accent">
          ⭐ {{ heroItem.rating.toFixed(1) }}
        </span>
        <span v-for="g in heroItem.genres?.slice(0, 3)" :key="g" class="feed-hero__chip">{{ g }}</span>
      </div>
      <p v-if="heroItem.overview" class="feed-hero__overview">{{ heroItem.overview }}</p>
      <div class="feed-hero__actions">
        <button
          v-if="heroItem.media_id && !heroItem.external"
          class="btn mh-btn mh-btn--primary"
          @click="$emit('play', heroItem)"
        >
          ▶ 播放
        </button>
        <button class="btn mh-btn mh-btn--secondary" @click="$emit('open', heroItem)">
          ℹ 详情
        </button>
        <button
          v-if="rowAction"
          class="btn mh-btn mh-btn--secondary"
          @click="onRowAction"
        >
          {{ actionCta }}
        </button>
      </div>
    </div>
  </section>

  <section v-else-if="row.type === 'divider'" class="feed-divider">
    <span v-if="row.title" class="feed-divider__label">{{ row.title }}</span>
    <hr class="feed-divider__line" />
  </section>

  <section v-else-if="row.type === 'text-banner'" class="feed-text-banner">
    <div
      class="feed-text-banner__inner"
      :class="{ 'feed-text-banner__inner--clickable': !!rowAction }"
      @click="onRowAction"
    >
      <div class="feed-text-banner__icon" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
          <circle cx="12" cy="12" r="3" />
          <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
        </svg>
      </div>
      <div class="feed-text-banner__body">
        <h3 v-if="row.title">{{ row.title.replace(/^📡\s*/, '') }}</h3>
        <p v-if="row.subtitle">{{ row.subtitle }}</p>
        <span v-if="rowAction" class="feed-text-banner__cta">{{ actionCta }}</span>
      </div>
    </div>
  </section>

  <section v-else-if="row.type === 'ranking'" class="feed-ranking">
    <header v-if="row.title || row.subtitle || rowAction" class="feed-ranking__header feed-section-header">
      <div class="feed-section-header__text">
        <h2 v-if="row.title" class="feed-section-header__title">{{ row.title }}</h2>
        <p v-if="row.subtitle" class="feed-section-header__subtitle">{{ row.subtitle }}</p>
      </div>
      <button v-if="rowAction" type="button" class="feed-section-header__link" @click="onRowAction">
        {{ actionCta }}
      </button>
    </header>
    <div class="feed-ranking__track">
      <article
        v-for="(item, index) in row.items"
        :key="item.external ? `tmdb-${item.tmdb_id}` : item.media_id"
        class="feed-ranking__tile"
        :class="{ 'feed-ranking__tile--top': index < 3 }"
        @click="$emit('open', item)"
      >
        <div class="feed-ranking__visual">
          <span class="feed-ranking__num">{{ index + 1 }}</span>
          <div class="feed-ranking__thumb">
            <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
            <span v-else class="feed-card__placeholder">{{ item.title.slice(0, 2) }}</span>
            <div class="feed-ranking__overlay">
              <span v-if="item.rating > 0" class="feed-ranking__score">⭐ {{ item.rating.toFixed(1) }}</span>
            </div>
          </div>
        </div>
        <div class="feed-ranking__caption">
          <h3 class="feed-ranking__name">{{ item.title }}</h3>
          <p class="feed-ranking__meta">
            <span v-if="item.year">{{ item.year }}</span>
            <span v-for="g in item.genres?.slice(0, 2)" :key="g">{{ g }}</span>
          </p>
        </div>
      </article>
    </div>
  </section>

  <section
    v-else-if="row.type === 'topic' && isTopicImmersive"
    class="feed-topic-immersive"
    :style="topicImmersiveBg"
  >
    <div class="feed-topic-immersive__overlay" />
    <div class="feed-topic-immersive__head">
      <h2 v-if="row.title" class="feed-topic-immersive__title">{{ row.title }}</h2>
      <p v-if="row.subtitle" class="feed-topic-immersive__subtitle">{{ row.subtitle }}</p>
      <button v-if="rowAction" type="button" class="feed-topic-immersive__link" @click="onRowAction">
        {{ actionCta }}
      </button>
    </div>
    <div
      class="feed-row__cards feed-row__cards--wrap card-style-landscape feed-topic-immersive__cards"
    >
      <article
        v-for="item in row.items"
        :key="item.external ? `tmdb-${item.tmdb_id}` : item.media_id"
        class="feed-card"
        @click="$emit('open', item)"
      >
        <div class="feed-card__poster">
          <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
          <span v-else class="feed-card__placeholder">{{ item.title.slice(0, 2) }}</span>
          <span v-if="item.rating > 0" class="feed-card__rating">⭐ {{ item.rating.toFixed(1) }}</span>
        </div>
        <div class="feed-card__title">{{ item.title }}</div>
      </article>
    </div>
  </section>

  <section v-else class="feed-row" :class="`feed-row--${row.type}`">
    <header v-if="row.title || row.subtitle || rowAction" class="feed-row__header feed-section-header">
      <div class="feed-section-header__text">
        <h2 v-if="row.title" class="feed-section-header__title">{{ row.title }}</h2>
        <p v-if="row.subtitle" class="feed-section-header__subtitle">{{ row.subtitle }}</p>
      </div>
      <button v-if="rowAction" type="button" class="feed-section-header__link" @click="onRowAction">
        {{ actionCta }}
      </button>
    </header>

    <div
      class="feed-row__cards"
      :class="[
        isGrid ? 'feed-row__cards--grid' : 'feed-row__cards--wrap',
        `card-style-${row.card_style || defaultCardStyle}`,
      ]"
    >
      <article
        v-for="item in row.items"
        :key="item.external ? `tmdb-${item.tmdb_id}` : item.media_id"
        class="feed-card"
        :class="{ 'feed-card--external': item.external }"
        @click="$emit('open', item)"
      >
        <div class="feed-card__poster">
          <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
          <span v-else class="feed-card__placeholder">{{ item.title.slice(0, 2) }}</span>
          <div class="feed-card__hover">
            <span class="feed-card__play">▶</span>
          </div>
          <span v-if="item.external" class="feed-card__badge">TMDB</span>
          <div v-if="item.progress && item.progress > 0" class="feed-card__progress">
            <div class="feed-card__progress-fill" :style="{ width: progressPct(item) + '%' }" />
          </div>
          <span v-if="item.progress && item.progress > 0" class="feed-card__resume">继续</span>
          <span v-if="item.rating > 0" class="feed-card__rating">⭐ {{ item.rating.toFixed(1) }}</span>
        </div>
        <div class="feed-card__title">{{ item.title }}</div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { FeedItem, FeedRow } from '@/api'
import {
  parseFeedAction,
  actionLabel,
  runFeedAction,
} from '@/utils/feedAction'

const props = defineProps<{
  row: FeedRow
  immersive?: boolean
}>()

defineEmits<{
  open: [item: FeedItem]
  play: [item: FeedItem]
}>()

const router = useRouter()

const rowAction = computed(() => parseFeedAction(props.row.config))
const actionCta = computed(() => actionLabel(rowAction.value, '查看更多 →'))

const isTopicImmersive = computed(
  () => props.row.type === 'topic' && props.row.config?.display === 'immersive',
)

const topicImmersiveBg = computed(() => {
  const first = props.row.items?.[0]
  const url = first?.backdrop_url || first?.poster_url
  return url ? { backgroundImage: `url(${url})` } : {}
})

const heroItem = computed<FeedItem | null>(() => {
  const playable = (items: FeedItem[]) =>
    items.find((i) => !i.external && i.media_id) || items[0]
  if (props.row.type !== 'hero-banner' || !props.row.items?.length) return null
  return playable(props.row.items)
})

const heroBg = computed(() => {
  const url = heroItem.value?.backdrop_url || heroItem.value?.poster_url
  return url ? { backgroundImage: `url(${url})` } : {}
})

const isGrid = computed(() => props.row.type === 'category-grid')

const defaultCardStyle = computed(() =>
  props.row.type === 'topic' ? 'landscape' : 'poster',
)

function onRowAction() {
  const action = rowAction.value
  if (!action) return
  runFeedAction(action, router)
}

function progressPct(item: FeedItem) {
  if (!item.progress || !item.duration) return 0
  return Math.min(100, (item.progress / item.duration) * 100)
}
</script>

<style lang="scss" scoped>
// ── 分区标题（榜单 / 货架 / 网格共用） ──
.feed-section-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--mh-space-4);
  margin-bottom: var(--mh-space-5);
  padding: 0 var(--mh-page-gutter);

  &__text {
    min-width: 0;
  }

  &__title {
    margin: 0;
    font-size: clamp(20px, 2.4vw, 26px);
    font-weight: 700;
    font-family: var(--mh-font-display);
    letter-spacing: -0.02em;
    display: flex;
    align-items: center;
    gap: var(--mh-space-3);

    &::before {
      content: '';
      flex-shrink: 0;
      width: 4px;
      height: 0.85em;
      border-radius: 2px;
      background: linear-gradient(180deg, var(--mh-primary), var(--mh-secondary));
    }
  }

  &__subtitle {
    margin: 6px 0 0 calc(4px + var(--mh-space-3));
    font-size: 14px;
    color: var(--mh-text-muted);
    line-height: 1.4;
  }

  &__link {
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--mh-outline);
    color: var(--mh-text-secondary);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    padding: 8px 14px;
    border-radius: var(--mh-radius-full);
    transition: background var(--mh-duration) var(--mh-ease),
                color var(--mh-duration) var(--mh-ease),
                border-color var(--mh-duration) var(--mh-ease);

    &:hover {
      background: rgba(108, 99, 255, 0.15);
      border-color: rgba(108, 99, 255, 0.35);
      color: var(--mh-text);
    }
  }
}

// ── Hero ──
.feed-hero {
  position: relative;
  height: min(75vh, 820px);
  min-height: 480px;
  margin-top: var(--mh-topbar-height);
  background-size: cover;
  background-position: center top;
  display: flex;
  align-items: center;
  padding: 0 var(--mh-page-gutter);

  &__overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      105deg,
      rgba(10, 10, 18, 0.97) 0%,
      rgba(10, 10, 18, 0.72) 38%,
      rgba(10, 10, 18, 0.25) 68%,
      rgba(10, 10, 18, 0.55) 100%
    );
  }

  &__content {
    position: relative;
    z-index: 1;
    max-width: 580px;
    color: var(--mh-text);
  }

  &__eyebrow {
    margin: 0 0 10px;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--mh-secondary);
  }

  &__title {
    font-size: clamp(32px, 4.5vw, 52px);
    font-weight: 800;
    font-family: var(--mh-font-display);
    margin: 0 0 var(--mh-space-4);
    line-height: 1.06;
    letter-spacing: -0.03em;
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--mh-space-2);
    margin-bottom: var(--mh-space-5);
  }

  &__chip {
    display: inline-flex;
    align-items: center;
    padding: 4px 10px;
    border-radius: var(--mh-radius-full);
    font-size: 12px;
    font-weight: 600;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid var(--mh-outline);
    color: var(--mh-text-secondary);
    backdrop-filter: blur(8px);

    &--accent {
      background: rgba(108, 99, 255, 0.2);
      border-color: rgba(108, 99, 255, 0.35);
      color: #c4c0ff;
    }
  }

  &__overview {
    font-size: 15px;
    line-height: 1.7;
    color: var(--mh-text-secondary);
    margin-bottom: var(--mh-space-6);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  &__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--mh-space-3);
  }

  &--immersive {
    height: min(92vh, 980px);
    min-height: 560px;
    align-items: flex-end;
    padding-bottom: clamp(var(--mh-space-10), 8vh, 80px);

    .feed-hero__overlay {
      background: linear-gradient(
        180deg,
        rgba(10, 10, 18, 0.15) 0%,
        rgba(10, 10, 18, 0.45) 40%,
        rgba(10, 10, 18, 0.98) 100%
      );
    }

    .feed-hero__content {
      max-width: 680px;
    }

    .feed-hero__title {
      font-size: clamp(38px, 5.5vw, 60px);
    }
  }
}

// ── 分隔线 → 区块标题 ──
.feed-divider {
  display: flex;
  align-items: center;
  margin: var(--mh-space-10) 0 var(--mh-space-5);
  padding: 0 var(--mh-page-gutter);

  &__label {
    font-size: clamp(18px, 2.2vw, 22px);
    font-weight: 700;
    font-family: var(--mh-font-display);
    letter-spacing: -0.02em;
    color: var(--mh-text);
    white-space: nowrap;
    display: flex;
    align-items: center;
    gap: var(--mh-space-3);

    &::before {
      content: '';
      width: 4px;
      height: 0.85em;
      border-radius: 2px;
      background: linear-gradient(180deg, var(--mh-primary), var(--mh-secondary));
    }
  }

  &__line {
    flex: 1;
    border: none;
    height: 1px;
    margin: 0 0 0 var(--mh-space-4);
    background: linear-gradient(90deg, var(--mh-outline-strong), transparent);
  }
}

// ── 功能横幅（直播等） ──
.feed-text-banner {
  margin: var(--mh-space-8) 0;
  padding: 0 var(--mh-page-gutter);

  &__inner {
    display: flex;
    align-items: center;
    gap: var(--mh-space-5);
    padding: var(--mh-space-5) var(--mh-space-6);
    border-radius: var(--mh-radius-lg);
    background: linear-gradient(
      135deg,
      rgba(108, 99, 255, 0.14) 0%,
      rgba(62, 207, 207, 0.08) 100%
    );
    border: 1px solid rgba(108, 99, 255, 0.22);
    box-shadow: var(--mh-shadow-md), inset 0 1px 0 rgba(255, 255, 255, 0.04);
  }

  &__icon {
    flex-shrink: 0;
    width: 52px;
    height: 52px;
    border-radius: var(--mh-radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(108, 99, 255, 0.2);
    color: var(--mh-primary);
    border: 1px solid rgba(108, 99, 255, 0.3);

    svg {
      width: 26px;
      height: 26px;
    }
  }

  &__body {
    min-width: 0;

    h3 {
      margin: 0 0 4px;
      font-size: 17px;
      font-weight: 700;
      font-family: var(--mh-font-display);
    }

    p {
      margin: 0;
      color: var(--mh-text-secondary);
      font-size: 14px;
      line-height: 1.5;
    }
  }

  &__inner--clickable {
    cursor: pointer;
    transition: transform 0.25s var(--mh-ease-spring),
                border-color 0.25s,
                box-shadow 0.25s;

    &:hover {
      transform: translateY(-2px);
      border-color: rgba(108, 99, 255, 0.45);
      box-shadow: var(--mh-shadow-lg), 0 0 40px rgba(108, 99, 255, 0.12);
    }
  }

  &__cta {
    display: inline-flex;
    align-items: center;
    margin-top: 10px;
    font-size: 13px;
    font-weight: 600;
    color: var(--mh-secondary);
  }
}

// ── 榜单：Netflix 式横向 TOP ──
.feed-ranking {
  margin-bottom: var(--mh-space-10);

  &__header {
    margin-bottom: var(--mh-space-4);
  }

  &__track {
    display: flex;
    gap: clamp(var(--mh-space-4), 2vw, var(--mh-space-6));
    overflow-x: auto;
    padding: var(--mh-space-2) var(--mh-page-gutter) var(--mh-space-4);
    scroll-snap-type: x proximity;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: thin;

    &::-webkit-scrollbar {
      height: 5px;
    }

    &::-webkit-scrollbar-thumb {
      background: rgba(255, 255, 255, 0.12);
      border-radius: 3px;
    }
  }

  &__tile {
    flex: 0 0 auto;
    width: clamp(150px, 16vw, 200px);
    scroll-snap-align: start;
    cursor: pointer;

    &--top .feed-ranking__num {
      background: linear-gradient(
        180deg,
        rgba(255, 255, 255, 0.98) 0%,
        rgba(108, 99, 255, 0.55) 100%
      );
      -webkit-background-clip: text;
      background-clip: text;
    }
  }

  &__visual {
    position: relative;
    display: flex;
    align-items: flex-end;
    height: clamp(170px, 20vw, 240px);
    margin-bottom: var(--mh-space-3);
  }

  &__num {
    position: absolute;
    left: 0;
    bottom: 0;
    z-index: 2;
    font-size: clamp(88px, 11vw, 128px);
    font-weight: 900;
    line-height: 0.82;
    font-family: var(--mh-font-display);
    letter-spacing: -0.06em;
    background: linear-gradient(
      180deg,
      rgba(255, 255, 255, 0.55) 0%,
      rgba(255, 255, 255, 0.12) 100%
    );
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    user-select: none;
    pointer-events: none;
  }

  &__thumb {
    position: relative;
    margin-left: clamp(36px, 5vw, 52px);
    width: clamp(96px, 11vw, 132px);
    height: 100%;
    border-radius: var(--mh-radius-md);
    overflow: hidden;
    background: var(--mh-surface-variant);
    border: 1px solid var(--mh-outline);
    box-shadow: var(--mh-shadow-md);
    transition: transform 0.28s var(--mh-ease-spring),
                box-shadow 0.28s;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(180deg, transparent 55%, rgba(0, 0, 0, 0.75) 100%);
    opacity: 0;
    transition: opacity 0.25s;
    display: flex;
    align-items: flex-end;
    padding: 8px;
  }

  &__score {
    font-size: 11px;
    font-weight: 600;
  }

  &__tile:hover {
    .feed-ranking__thumb {
      transform: scale(1.05) translateY(-4px);
      box-shadow: var(--mh-shadow-lg), var(--mh-shadow-glow);
    }

    .feed-ranking__overlay {
      opacity: 1;
    }
  }

  &__caption {
    padding-left: clamp(36px, 5vw, 52px);
  }

  &__name {
    margin: 0 0 4px;
    font-size: 14px;
    font-weight: 600;
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  &__meta {
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 4px 8px;
    font-size: 11px;
    color: var(--mh-text-muted);
  }
}

// ── 货架 / 网格 ──
.feed-row {
  margin-bottom: var(--mh-space-10);

  &__header {
    margin-bottom: var(--mh-space-5);
  }

  &__cards--wrap {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
    gap: var(--mh-space-4) var(--mh-space-5);
    padding: 0 var(--mh-page-gutter) var(--mh-space-2);
  }

  &__cards--grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
    gap: var(--mh-space-4) var(--mh-space-5);
    padding: 0 var(--mh-page-gutter);
  }
}

.feed-card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease-spring);
  min-width: 0;

  &:hover {
    transform: translateY(-4px);
    z-index: 10;

    .feed-card__poster {
      box-shadow: var(--mh-shadow-lg), var(--mh-shadow-glow);
      border-color: rgba(108, 99, 255, 0.3);

      img {
        transform: scale(1.06);
      }
    }

    .feed-card__hover {
      opacity: 1;
    }
  }

  &__poster {
    position: relative;
    aspect-ratio: 2/3;
    background: linear-gradient(145deg, var(--mh-surface-variant), var(--mh-bg));
    border-radius: var(--mh-radius-md);
    overflow: hidden;
    border: 1px solid var(--mh-outline);
    transition: box-shadow var(--mh-duration) var(--mh-ease),
                border-color var(--mh-duration) var(--mh-ease);

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
      transition: transform 0.35s var(--mh-ease);
    }
  }

  &__hover {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(10, 10, 18, 0.45);
    opacity: 0;
    transition: opacity 0.25s;
  }

  &__play {
    width: 44px;
    height: 44px;
    border-radius: var(--mh-radius-full);
    background: rgba(255, 255, 255, 0.95);
    color: var(--mh-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    padding-left: 3px;
    box-shadow: var(--mh-shadow-md);
  }

  &__placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    font-size: 28px;
    font-weight: 700;
    color: rgba(255, 255, 255, 0.25);
  }

  &__title {
    margin-top: var(--mh-space-3);
    font-size: 13px;
    font-weight: 500;
    line-height: 1.4;
    color: var(--mh-text-secondary);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    transition: color var(--mh-duration);

    .feed-card:hover & {
      color: var(--mh-text);
    }
  }

  &__badge {
    position: absolute;
    top: 8px;
    right: 8px;
    padding: 2px 8px;
    border-radius: 6px;
    background: rgba(108, 99, 255, 0.92);
    color: #fff;
    font-size: 10px;
    font-weight: 700;
    z-index: 2;
  }

  &__rating {
    position: absolute;
    bottom: 8px;
    left: 8px;
    padding: 3px 8px;
    border-radius: var(--mh-radius-full);
    background: rgba(0, 0, 0, 0.72);
    font-size: 11px;
    font-weight: 600;
    z-index: 2;
    backdrop-filter: blur(4px);
  }

  &__resume {
    position: absolute;
    top: 8px;
    left: 8px;
    padding: 3px 8px;
    border-radius: var(--mh-radius-full);
    background: var(--mh-primary);
    color: #fff;
    font-size: 10px;
    font-weight: 700;
    z-index: 2;
  }

  &__progress {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: rgba(255, 255, 255, 0.2);
    z-index: 2;
  }

  &__progress-fill {
    height: 100%;
    background: var(--mh-primary);
  }
}

.card-style-landscape .feed-card__poster {
  aspect-ratio: 16/9;
}

.card-style-landscape.feed-row__cards--wrap {
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
}

.card-style-square .feed-card__poster {
  aspect-ratio: 1/1;
}

.card-style-banner .feed-card__poster {
  aspect-ratio: 21/9;
}

.card-style-banner.feed-row__cards--wrap {
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
}

.feed-row--topic .feed-row__cards--wrap {
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
}

// ── 沉浸式专题 ──
.feed-topic-immersive {
  position: relative;
  margin: var(--mh-space-10) var(--mh-page-gutter);
  padding: var(--mh-space-10) var(--mh-space-8) var(--mh-space-8);
  background-size: cover;
  background-position: center;
  border-radius: var(--mh-radius-lg);
  overflow: hidden;
  border: 1px solid var(--mh-outline);

  &__overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      180deg,
      rgba(10, 10, 18, 0.4) 0%,
      rgba(10, 10, 18, 0.88) 50%,
      rgba(10, 10, 18, 0.98) 100%
    );
  }

  &__head {
    position: relative;
    z-index: 1;
    margin-bottom: var(--mh-space-6);
    max-width: 560px;
  }

  &__title {
    margin: 0 0 8px;
    font-size: clamp(26px, 3.5vw, 36px);
    font-weight: 800;
    font-family: var(--mh-font-display);
    letter-spacing: -0.02em;
  }

  &__subtitle {
    margin: 0 0 var(--mh-space-4);
    color: var(--mh-text-secondary);
    font-size: 15px;
    line-height: 1.55;
  }

  &__link {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid var(--mh-outline-strong);
    color: var(--mh-text);
    padding: 10px 18px;
    border-radius: var(--mh-radius-full);
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    transition: background 0.2s, border-color 0.2s;

    &:hover {
      background: rgba(108, 99, 255, 0.2);
      border-color: rgba(108, 99, 255, 0.4);
    }
  }

  &__cards {
    position: relative;
    z-index: 1;
    padding: 0 !important;
  }
}

@media (max-width: 640px) {
  .feed-hero {
    min-height: 420px;
    align-items: flex-end;
    padding-bottom: var(--mh-space-8);

    &__overview {
      -webkit-line-clamp: 2;
    }
  }

  .feed-text-banner__inner {
    flex-direction: column;
    align-items: flex-start;
  }

  .feed-ranking__track {
    scroll-padding-left: var(--mh-page-gutter);
  }
}
</style>
