package com.mediahub.tv.data.model

import kotlinx.serialization.Serializable

/**
 * 媒资条目（FeedItem）
 */
@Serializable
data class MediaItem(
    val mediaId: String,
    val title: String,
    val year: Int? = null,
    val posterUrl: String? = null,
    val backdropUrl: String? = null,
    val rating: Double = 0.0,
    val type: String = "movie",
    val overview: String? = null,
    val duration: Int? = null,
    val genres: List<String> = emptyList(),
    val progress: Int? = null,
)

/**
 * Feed Row（横滑一行）
 */
@Serializable
data class FeedRow(
    val id: String,
    val type: String,
    val title: String? = null,
    val subtitle: String? = null,
    val cardStyle: String? = null,
    val items: List<MediaItem> = emptyList(),
)

/**
 * Feed（完整布局）
 */
@Serializable
data class Feed(
    val version: Int,
    val platform: String,
    val updatedAt: String,
    val rows: List<FeedRow>,
)

/**
 * 媒资详情
 */
@Serializable
data class MediaDetail(
    val id: String,
    val title: String,
    val originalTitle: String? = null,
    val year: Int? = null,
    val type: String,
    val rating: Double = 0.0,
    val posterUrl: String? = null,
    val backdropUrl: String? = null,
    val overview: String? = null,
    val runtime: Int? = null,
    val genres: List<String> = emptyList(),
    val hasSubtitle: Boolean = false,
    val videoCodec: String? = null,
    val audioCodec: String? = null,
    val resolution: String? = null,
    val storagePath: String,
)

/**
 * Profile
 */
@Serializable
data class Profile(
    val id: String,
    val userId: String,
    val name: String,
    val avatarUrl: String? = null,
    val isKid: Boolean = false,
    val avatarEmoji: String? = null,
) {
    /** 用于 UI 显示的头像（emoji 优先，没有就用首字母） */
    val avatar_emoji: String
        get() = avatarEmoji
            ?: if (isKid) "🧒" else when (name.firstOrNull()?.uppercaseChar()) {
                'A', 'B', 'C' -> "🦊"
                'D', 'E', 'F' -> "🐼"
                'G', 'H', 'I' -> "🦁"
                'J', 'K', 'L' -> "🐯"
                'M', 'N', 'O' -> "🐰"
                'P', 'Q', 'R' -> "🐨"
                'S', 'T', 'U' -> "🐻"
                else -> "🐶"
            }

    /** 别名：display_name -> name（保持向后兼容） */
    val display_name: String get() = name
}
