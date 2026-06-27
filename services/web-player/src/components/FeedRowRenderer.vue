<template>
  <section v-if="row.type === 'divider'" class="feed-divider">
    <span v-if="row.title" class="feed-divider__label">{{ row.title }}</span>
    <hr class="feed-divider__line" />
  </section>

  <section v-else-if="row.type === 'text-banner'" class="feed-text-banner">
    <div class="feed-text-banner__inner">
      <h3 v-if="row.title">{{ row.title }}</h3>
      <p v-if="row.subtitle">{{ row.subtitle }}</p>
    </div>
  </section>

  <section v-else class="feed-row" :class="`feed-row--${row.type}`">
    <header v-if="row.title || row.subtitle" class="feed-row__header">
      <h2 class="feed-row__title">{{ row.title }}</h2>
      <span v-if="row.subtitle" class="feed-row__subtitle">{{ row.subtitle }}</span>
    </header>

    <div
      class="feed-row__cards"
      :class="[
        isGrid ? 'feed-row__cards--grid' : 'feed-row__cards--scroll',
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
import type { FeedItem, FeedRow } from '@/api'

const props = defineProps<{
  row: FeedRow
}>()

defineEmits<{
  open: [item: FeedItem]
}>()

const isGrid = computed(() => props.row.type === 'category-grid')

const defaultCardStyle = computed(() =>
  props.row.type === 'topic' ? 'landscape' : 'poster',
)

function progressPct(item: FeedItem) {
  if (!item.progress || !item.duration) return 0
  return Math.min(100, (item.progress / item.duration) * 100)
}
</script>

<style lang="scss" scoped>
.feed-divider {
  display: flex;
  align-items: center;
  gap: var(--mh-space-4);
  margin: var(--mh-space-2) 0;

  &__label {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--mh-text-muted);
    white-space: nowrap;
  }

  &__line {
    flex: 1;
    border: none;
    border-top: 1px solid var(--mh-outline);
    margin: 0;
  }
}

.feed-text-banner {
  margin: var(--mh-space-2) 0;

  &__inner {
    padding: var(--mh-space-5) var(--mh-space-6);
    border-radius: var(--mh-radius-md);
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.18), rgba(168, 85, 247, 0.12));
    border: 1px solid rgba(99, 102, 241, 0.25);

    h3 {
      margin: 0 0 6px;
      font-size: 18px;
      font-weight: 600;
    }

    p {
      margin: 0;
      color: var(--mh-text-secondary);
      font-size: 14px;
      line-height: 1.5;
    }
  }
}

.feed-row {
  margin-bottom: var(--mh-space-6);

  &__header {
    display: flex;
    align-items: baseline;
    gap: var(--mh-space-3);
    margin-bottom: var(--mh-space-4);
    padding: 0 clamp(var(--mh-space-4), 4vw, var(--mh-space-6));
  }

  &__title {
    margin: 0;
    font-size: clamp(18px, 2vw, 22px);
    font-weight: 700;
    font-family: var(--mh-font-display);
  }

  &__subtitle {
    font-size: 14px;
    color: var(--mh-text-muted);
  }

  &__cards--scroll {
    display: flex;
    gap: var(--mh-space-4);
    overflow-x: auto;
    padding: 0 clamp(var(--mh-space-4), 4vw, var(--mh-space-6)) var(--mh-space-4);
    scroll-behavior: smooth;
    scrollbar-width: thin;

    &::-webkit-scrollbar {
      height: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: rgba(255, 255, 255, 0.15);
      border-radius: 3px;
    }
  }

  &__cards--grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: var(--mh-space-4);
    padding: 0 clamp(var(--mh-space-4), 4vw, var(--mh-space-6));
  }
}

.feed-card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease-spring);

  .feed-row__cards--scroll & {
    flex-shrink: 0;
    width: 200px;
  }

  &:hover {
    transform: scale(1.04);
    z-index: 10;

    .feed-card__poster {
      box-shadow: var(--mh-shadow-lg), var(--mh-shadow-glow);
    }
  }

  &__poster {
    position: relative;
    aspect-ratio: 2/3;
    background: linear-gradient(145deg, var(--mh-surface-variant), var(--mh-bg));
    border-radius: var(--mh-radius-md);
    overflow: hidden;
    border: 1px solid var(--mh-outline);
    transition: box-shadow var(--mh-duration) var(--mh-ease);

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
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
    margin-top: var(--mh-space-2);
    font-size: 13px;
    font-weight: 500;
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
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
  }

  &__rating {
    position: absolute;
    bottom: 8px;
    left: 8px;
    padding: 2px 6px;
    border-radius: 6px;
    background: rgba(0, 0, 0, 0.65);
    font-size: 11px;
  }

  &__resume {
    position: absolute;
    top: 8px;
    left: 8px;
    padding: 2px 8px;
    border-radius: 6px;
    background: var(--mh-primary);
    color: #fff;
    font-size: 10px;
    font-weight: 700;
  }

  &__progress {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: rgba(255, 255, 255, 0.2);
  }

  &__progress-fill {
    height: 100%;
    background: var(--mh-primary);
  }
}

.card-style-landscape .feed-card__poster {
  aspect-ratio: 16/9;
}

.card-style-landscape.feed-row__cards--scroll .feed-card {
  width: 280px;
}

.card-style-square .feed-card__poster {
  aspect-ratio: 1/1;
}

.card-style-banner .feed-card__poster {
  aspect-ratio: 21/9;
}

.card-style-banner.feed-row__cards--scroll .feed-card {
  width: 360px;
}

.feed-row--topic .feed-row__cards--scroll .feed-card {
  width: 300px;
}
</style>
