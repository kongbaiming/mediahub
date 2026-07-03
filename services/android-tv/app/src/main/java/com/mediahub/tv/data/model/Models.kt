package com.mediahub.tv.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * 媒资条目（FeedItem）
 */
@Serializable
data class MediaItem(
    @SerialName("media_id") val mediaId: String,
    val title: String,
    val year: Int? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("backdrop_url") val backdropUrl: String? = null,
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
    @SerialName("card_style") val cardStyle: String? = null,
    val items: List<MediaItem> = emptyList(),
)

/**
 * Feed（完整布局）
 */
@Serializable
data class Feed(
    val version: Int,
    val platform: String,
    @SerialName("updated_at") val updatedAt: String,
    val rows: List<FeedRow>,
)

/**
 * 媒资详情
 */
@Serializable
data class MediaDetail(
    val id: String,
    val title: String,
    @SerialName("original_title") val originalTitle: String? = null,
    val year: Int? = null,
    val type: String,
    val kind: String = "single",
    val rating: Double = 0.0,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("backdrop_url") val backdropUrl: String? = null,
    val overview: String? = null,
    val runtime: Int? = null,
    val genres: List<String> = emptyList(),
    @SerialName("has_subtitle") val hasSubtitle: Boolean = false,
    @SerialName("video_codec") val videoCodec: String? = null,
    @SerialName("audio_codec") val audioCodec: String? = null,
    val resolution: String? = null,
    @SerialName("storage_path") val storagePath: String,
    val seasons: List<Season>? = null,
) {
    val isSeries: Boolean get() = kind == "series"
}

/**
 * Profile
 *
 * 字段命名走 @SerialName 而不是手写 alias 字段，
 * 与服务端 snake_case 对齐（display_name / user_id / avatar_url 等）。
 */
@Serializable
data class Profile(
    val id: String,
    @SerialName("user_id") val userId: String,
    @SerialName("display_name") val name: String,
    @SerialName("avatar_url") val avatarUrl: String? = null,
    @SerialName("is_kid") val isKid: Boolean = false,
    @SerialName("avatar_emoji") val avatarEmoji: String? = null,
) {
    /**
     * UI 显示用的头像：优先用服务端返回的 emoji，
     * 否则按名字首字母给一个 fallback。
     */
    val avatarEmojiResolved: String
        get() = avatarEmoji ?: pickEmoji(name, isKid)

    private fun pickEmoji(name: String, isKid: Boolean): String {
        if (isKid) return KID_EMOJI
        return when (name.firstOrNull()?.uppercaseChar()) {
            'A', 'B', 'C' -> "\uD83E\uDD8A"   // 狐狸
            'D', 'E', 'F' -> "\uD83D\uDC3C"   // 熊猫
            'G', 'H', 'I' -> "\uD83E\uDD81"   // 狮子
            'J', 'K', 'L' -> "\uD83D\uDC2F"   // 老虎
            'M', 'N', 'O' -> "\uD83D\uDC30"   // 兔子
            'P', 'Q', 'R' -> "\uD83D\uDC28"   // 考拉
            'S', 'T', 'U' -> "\uD83D\uDC3B"   // 熊
            else -> "\uD83D\uDC36"            // 狗
        }
    }

    private companion object {
        const val KID_EMOJI = "\uD83D\uDC78" // 小孩
    }
}
/**
 * 季
 */
@Serializable
data class Season(
    val id: String,
    @SerialName("season_number") val seasonNumber: Int,
    val title: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("episode_count") val episodeCount: Int = 0,
    val episodes: List<Episode> = emptyList(),
)

/**
 * 集
 */
@Serializable
data class Episode(
    val id: String,
    @SerialName("episode_number") val episodeNumber: Int,
    val title: String? = null,
    val duration: Int = 0,
    @SerialName("file_path") val filePath: String? = null,
    @SerialName("still_url") val stillUrl: String? = null,
)

/**
 * 字幕轨
 */
@Serializable
data class SubtitleTrack(
    val id: String,
    val language: String = "zh",
    val format: String = "srt",
    val label: String? = null,
    val source: String = "manual",
    @SerialName("is_default") val isDefault: Boolean = false,
)
